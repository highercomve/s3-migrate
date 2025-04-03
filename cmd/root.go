/*
Copyright © 2025 Sergio Marin <@highercomve>
*/
package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/highercomve/s3-migrate/lib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "s3-migrate",
	Short:         "Migrate objects from source bucket to destination bucket",
	Long:          `Migrate objects from source bucket to destination bucket based on MongoDB records`,
	RunE:          lib.MigrateStorage,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Config and profiling flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cobra.yaml)")
	rootCmd.PersistentFlags().String("cpuprofile", "", "CPU profiling")

	// Source bucket flags
	rootCmd.Flags().StringP("source-key", "k", "", "Source s3 ACCESS_KEY")
	rootCmd.Flags().StringP("source-secret", "s", "", "Source s3 SECRET")
	rootCmd.Flags().StringP("source-region", "r", "", "Source s3 REGION")
	rootCmd.Flags().StringP("source-bucket", "b", "", "Source s3 BUCKET")
	rootCmd.Flags().StringP("source-endpoint", "e", "", "Source s3 ENDPOINT")

	// Destination bucket flags
	rootCmd.Flags().StringP("dest-key", "K", "", "Destination s3 ACCESS_KEY")
	rootCmd.Flags().StringP("dest-secret", "S", "", "Destination s3 SECRET")
	rootCmd.Flags().StringP("dest-region", "R", "", "Destination s3 REGION")
	rootCmd.Flags().StringP("dest-bucket", "B", "", "Destination s3 BUCKET")
	rootCmd.Flags().StringP("dest-endpoint", "E", "", "Destination s3 ENDPOINT")

	// Database flags
	rootCmd.Flags().StringP("database", "d", "", "database name")
	rootCmd.Flags().StringP("collection", "c", "", "database collection")
	rootCmd.Flags().StringP("connection", "m", "", "database connection url")
	rootCmd.Flags().StringP("filter", "f", `{"sizeint":{"$gt": 0}}`, "database filter")

	// Performance flags
	rootCmd.Flags().Int64P("limit", "l", 100, "Request limit")
	rootCmd.Flags().Int64("ratelimit", 0, "rate limit per second to search for objects in s3")
	rootCmd.Flags().Int64("concurrency", 0, "concurrency level")

	viper.BindPFlags(rootCmd.Flags())
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		p, err := os.Executable()
		if err != nil {
			fmt.Println(err)
			return
		}
		cobra.CheckErr(err)

		// Search config in home directory with name ".config"
		viper.AddConfigPath(home)
		viper.AddConfigPath(path.Dir(p))
		viper.SetConfigType("yaml")
		viper.SetConfigName("s3-migrate")
	}

	viper.AutomaticEnv()
	viper.ReadInConfig()
}
