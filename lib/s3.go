/*
Copyright © 2025 Sergio Marin <@highercomve>
*/
package lib

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/schollz/progressbar/v3"
)

var endpoint string = "s3.amazonaws.com"

type S3ConnParams struct {
	Key      string
	Secret   string
	Region   string
	Bucket   string
	Endpoint string
}

type S3Client struct {
	client *minio.Client
	bucket string
}

func NewS3Connect(ctxP context.Context, params *S3ConnParams) (client *S3Client, err error) {
	s3Client := &S3Client{bucket: params.Bucket}
	if params.Endpoint != "" {
		endpoint = params.Endpoint
	}

	s3Client.client, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(params.Key, params.Secret, ""),
		Region: params.Region,
		Secure: true,
	})
	if err != nil {
		log.Fatalln(err)
	}

	return s3Client, err
}

func (s *S3Client) ObjectExist(ctx context.Context, id string) (exist bool, err error) {
	exist = false

	_, err = s.client.StatObject(ctx, s.bucket, id, minio.StatObjectOptions{})
	if err != nil && strings.Contains(err.Error(), "The specified key does not exist") {
		return false, nil
	}
	if err != nil {
		return
	}

	exist = true

	return exist, err
}

// CopyObject copies an object from source to destination bucket
func (s *S3Client) CopyObjectOld(ctx context.Context, source *S3Client, objectSHA string, dryRun bool) error {
	if dryRun {
		fmt.Println("Dry run enabled: no action taken")
		return nil
	}

	// Get object info for content type
	objectInfo, err := source.client.StatObject(ctx, source.bucket, objectSHA, minio.StatObjectOptions{})
	if err != nil {
		return fmt.Errorf("error getting object info: %v", err)
	}

	// Get object from source as a stream
	object, err := source.client.GetObject(ctx, source.bucket, objectSHA, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("error getting object from source: %v", err)
	}
	defer object.Close()

	// Put object in destination as a stream
	_, err = s.client.PutObject(ctx, s.bucket, objectSHA, object, objectInfo.Size, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("error putting object in destination: %v", err)
	}

	return nil
}

// CopyObject copies an object with progress bar
func (s *S3Client) CopyObject(ctx context.Context, source *S3Client, objectSHA string, dryRun bool) error {
	// Get object info for size
	objectInfo, err := source.client.StatObject(ctx, source.bucket, objectSHA, minio.StatObjectOptions{})
	if err != nil {
		return fmt.Errorf("error getting object info: %v", err)
	}

	// Create progress bar
	bar := progressbar.DefaultBytes(
		objectInfo.Size,
		fmt.Sprintf("Copying %s", objectSHA),
	)

	if dryRun {
		// For dry run, simulate the progress bar filling up
		for i := int64(0); i <= objectInfo.Size; i += objectInfo.Size / 100 {
			bar.Set64(i)
			time.Sleep(10 * time.Millisecond)
		}
		bar.Finish()
		fmt.Println(" (Dry run: no actual copy performed)")
		return nil
	}

	// For server-side copy, we need to simulate progress
	done := make(chan struct{})
	go func() {
		// Update progress bar until done
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				bar.Finish()
				return
			case <-ticker.C:
				// Check if copy is still in progress
				_, err := s.client.StatObject(ctx, s.bucket, objectSHA, minio.StatObjectOptions{})
				if err == nil {
					// Object exists in destination, copy is complete
					bar.Set64(objectInfo.Size)
					return
				}
				// Show some progress to indicate the copy is ongoing
				current := bar.State().CurrentBytes
				if current < float64(objectInfo.Size) {
					// Increment by ~5% of file size as a visual indicator
					increment := objectInfo.Size / 20
					if increment < 1 {
						increment = 1
					}
					newPos := int64(current) + increment
					if newPos > objectInfo.Size {
						newPos = objectInfo.Size
					}
					bar.Set64(newPos)
				}
			}
		}
	}()

	// Define source info
	srcOpts := minio.CopySrcOptions{
		Bucket: source.bucket,
		Object: objectSHA,
	}

	// Define destination info
	dstOpts := minio.CopyDestOptions{
		Bucket: s.bucket,
		Object: objectSHA,
	}

	// Perform server-side copy
	_, err = s.client.CopyObject(ctx, dstOpts, srcOpts)
	close(done) // Signal that the copy operation is done

	if err != nil {
		return fmt.Errorf("error copying object: %v", err)
	}

	return nil
}
