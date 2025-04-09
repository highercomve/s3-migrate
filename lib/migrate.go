/*
Copyright © 2025 Sergio Marin <@highercomve>
*/
package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/time/rate"
)

type S3MigrationParams struct {
	Source      *S3ConnParams
	Destination *S3ConnParams
	Database    string
	Collection  *mongo.Collection
	Connection  string
	Filter      bson.M
	Limit       int64
	RateLimit   int64
	DryRun      bool
	Concurrency int64
}

type Object struct {
	ID           string `json:"id" bson:"id"`
	StorageID    string `json:"storage-id" bson:"_id"`
	Owner        string `json:"owner"`
	ObjectName   string `json:"objectname"`
	Sha          string `json:"sha256sum"`
	Size         string `json:"size"`
	SizeInt      int64  `json:"sizeint"`
	MimeType     string `json:"mime-type"`
	initialized  bool
	LinkedObject string    `json:"-" bson:"linked_object"`
	TimeCreated  time.Time `json:"time-created" bson:"timecreated"`
	TimeModified time.Time `json:"time-modified" bson:"timemodified"`
}

type MigrationReport struct {
	TotalObjects         int64         `json:"total_objects"`
	Copied               int64         `json:"copied"`
	Skipped              int64         `json:"skipped"`
	AlreadyInDestination int64         `json:"already_in_destination"`
	Errors               int64         `json:"errors"`
	StartTime            time.Time     `json:"start_time"`
	EndTime              time.Time     `json:"end_time"`
	Duration             time.Duration `json:"duration"`
}

func MigrateStorage(cmd *cobra.Command, args []string) (err error) {
	// Source bucket params
	sourceKey := viper.GetString("source-key")
	sourceSecret := viper.GetString("source-secret")
	sourceRegion := viper.GetString("source-region")
	sourceBucket := viper.GetString("source-bucket")
	sourceEndpoint := viper.GetString("source-endpoint")

	// Destination bucket params
	destKey := viper.GetString("dest-key")
	destSecret := viper.GetString("dest-secret")
	destRegion := viper.GetString("dest-region")
	destBucket := viper.GetString("dest-bucket")
	destEndpoint := viper.GetString("dest-endpoint")

	// Database params
	database := viper.GetString("database")
	collectionName := viper.GetString("collection")
	connection := viper.GetString("connection")
	limit := viper.GetInt64("limit")
	ratelimit := viper.GetInt64("ratelimit")
	filterString := viper.GetString("filter")
	dryRun := viper.GetBool("dry-run")
	concurrency := viper.GetInt64("concurrency")
	if concurrency == 0 {
		concurrency = int64(runtime.NumCPU())
	}

	// Print configuration
	fmt.Printf("\nMigration Configuration:\n")
	fmt.Printf("Source: %s (%s, %s)\n", sourceBucket, sourceRegion, sourceEndpoint)
	fmt.Printf("Destination: %s (%s, %s)\n", destBucket, destRegion, destEndpoint)
	fmt.Printf("Database: %s\n", database)
	fmt.Printf("Collection: %s\n", collectionName)
	fmt.Printf("Connection: %s\n", connection)
	fmt.Printf("Filter: %s\n", filterString)
	fmt.Printf("Batch Size: %d\n", limit)
	if ratelimit > 0 {
		fmt.Printf("Rate Limit: %d ops/sec\n", ratelimit)
	}
	if dryRun {
		fmt.Printf("Dry Run: Enabled\n")
	}
	fmt.Printf("Concurrency Level: %d\n", concurrency)

	// CPU profiling
	cpuprofile := viper.GetString("cpuprofile")
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Parse filter
	filter := bson.M{}
	if err := json.Unmarshal([]byte(filterString), &filter); err != nil {
		log.Printf("Error parsing filter configuration: %v", err)
		return err
	}
	log.Printf("Successfully parsed filter configuration: %v", filter)

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to MongoDB
	// Log the attempt to connect to MongoDB
	log.Println("Attempting to connect to MongoDB...")
	storage, err := NewDbConnection(ctx, connection)
	if err != nil {
		return err
	}
	// Log successful connection
	log.Println("Successfully connected to MongoDB.")

	db := storage.GetDatabase(database)
	collection := db.Collection(collectionName)

	// Create S3 clients
	sourceParams := &S3ConnParams{
		Key:      sourceKey,
		Secret:   sourceSecret,
		Region:   sourceRegion,
		Bucket:   sourceBucket,
		Endpoint: sourceEndpoint,
	}

	destParams := &S3ConnParams{
		Key:      destKey,
		Secret:   destSecret,
		Region:   destRegion,
		Bucket:   destBucket,
		Endpoint: destEndpoint,
	}

	// Create rate limiter if specified
	var limiter *rate.Limiter
	if ratelimit > 0 {
		limiter = rate.NewLimiter(rate.Limit(ratelimit), 1)
	}

	// Get total count of documents
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return err
	}

	if count <= 0 {
		fmt.Println("Zero objects found to migrate. Exiting.")
		return nil
	}

	fmt.Printf("Found %d objects to migrate\n\n", count)

	// Create migration params
	migrationParams := &S3MigrationParams{
		Source:      sourceParams,
		Destination: destParams,
		Database:    database,
		Collection:  collection,
		Connection:  connection,
		Filter:      filter,
		Limit:       limit,
		RateLimit:   ratelimit,
		DryRun:      dryRun,
		Concurrency: concurrency,
	}

	// Start migration
	return migrateObjects(ctx, migrationParams, count, limiter)
}

func migrateObjects(ctx context.Context, params *S3MigrationParams, totalCount int64, limiter *rate.Limiter) error {
	startTime := time.Now()

	sourceClient, err := NewS3Connect(ctx, params.Source)
	if err != nil {
		return err
	}

	destClient, err := NewS3Connect(ctx, params.Destination)
	if err != nil {
		return err
	}

	// Create worker pool
	var wg sync.WaitGroup
	objectChan := make(chan Object, params.Concurrency)

	// Statistics tracking
	var copiedCount, skippedCount, alreadyExistsCount, errorCount int64

	// Start workers
	for i := int64(0); i < params.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for doc := range objectChan {
				// Apply rate limiting if configured
				if limiter != nil {
					if err := limiter.Wait(ctx); err != nil {
						log.Printf("Rate limiter error: %v", err)
						errorCount++
						continue
					}
				}

				// Get object SHA from document
				storageID := doc.StorageID
				// Check if object exists in source bucket
				exists, err := sourceClient.ObjectExist(ctx, storageID)
				if err != nil {
					log.Printf("Error checking object %s: %v", storageID, err)
					errorCount++
					continue
				}

				if !exists {
					log.Printf("Object %s not found in source bucket", storageID)
					skippedCount++
					continue
				}

				// Check if object exists in destination bucket with the same SHA
				destExists, err := destClient.ObjectExist(ctx, storageID)
				if err != nil {
					log.Printf("Error checking object %s in destination bucket: %v", storageID, err)
					errorCount++
					continue
				}

				if destExists {
					log.Printf("Object %s already exists in destination bucket with matching SHA. Skipping copy.", storageID)
					alreadyExistsCount++
					continue
				}

				if params.DryRun {
					log.Printf("Dry Run: Would copy object %s from source to destination", storageID)
					copiedCount++
					continue
				}

				// Copy object from source to destination
				if err := destClient.CopyObject(ctx, sourceClient, storageID, params.DryRun); err != nil {
					log.Printf("Error copying object %s: %v", storageID, err)
					errorCount++
					continue
				}

				copiedCount++
			}
		}()
	}

	// Fetch documents and send to workers
	cursor, err := params.Collection.Find(ctx, params.Filter, &options.FindOptions{})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var doc Object
		if err := cursor.Decode(&doc); err != nil {
			log.Printf("Error decoding document: %v", err)
			errorCount++
			continue
		}
		objectChan <- doc
	}

	close(objectChan)
	wg.Wait()

	endTime := time.Now()
	duration := time.Since(startTime)

	// Final summary report
	report := MigrationReport{
		TotalObjects:         totalCount,
		Copied:               copiedCount,
		Skipped:              skippedCount,
		AlreadyInDestination: alreadyExistsCount,
		Errors:               errorCount,
		StartTime:            startTime,
		EndTime:              endTime,
		Duration:             duration,
	}

	fmt.Println("\nMigration Summary Report:")
	fmt.Printf("Total Objects: %d\n", report.TotalObjects)
	fmt.Printf("Copied: %d\n", report.Copied)
	fmt.Printf("Skipped (not found in source): %d\n", report.Skipped)
	fmt.Printf("Already in Destination: %d\n", report.AlreadyInDestination)
	fmt.Printf("Errors: %d\n", report.Errors)
	fmt.Printf("Start Time: %s\n", report.StartTime)
	fmt.Printf("End Time: %s\n", report.EndTime)
	fmt.Printf("Duration: ")
	if report.Duration.Hours() >= 1 {
		fmt.Printf("%.2f hours\n", report.Duration.Hours())
	} else if report.Duration.Minutes() >= 1 {
		fmt.Printf("%.2f minutes\n", report.Duration.Minutes())
	} else if report.Duration.Seconds() >= 1 {
		fmt.Printf("%.2f seconds\n", report.Duration.Seconds())
	} else {
		fmt.Printf("%d milliseconds\n", report.Duration.Milliseconds())
	}

	fmt.Println("\nMigration completed!")
	return nil
}
