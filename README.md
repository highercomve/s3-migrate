# S3 Migrate

## Overview

The `s3-migrate` tool is designed to facilitate the migration of objects from a source S3 bucket to a destination S3 bucket based on records stored in a MongoDB database. This tool is particularly useful for scenarios where you need to move large amounts of data between S3 buckets while ensuring that only the relevant objects are migrated, as determined by the MongoDB records.

## Features

- **Source and Destination S3 Buckets**: Specify the source and destination S3 buckets, including their access keys, secrets, regions, and endpoints.
- **MongoDB Integration**: Connect to a MongoDB database to fetch records that determine which objects need to be migrated.
- **Filtering**: Apply filters to MongoDB queries to select specific records for migration.
- **Rate Limiting**: Control the rate of requests to the S3 service to avoid throttling.
- **Concurrency**: Set the level of concurrency for the migration process.
- **Dry Run**: Perform a dry run to see what would be migrated without actually performing the migration.
- **CPU Profiling**: Enable CPU profiling to analyze the performance of the migration process.
- **Progress Bar**: Visual feedback on the progress of each object being copied.

## Installation

To install the `s3-migrate` tool, clone the repository and build the project using Go:

```sh
git clone https://github.com/highercomve/s3-migrate.git
cd s3-migrate
go build -o s3-migrate
```

## Usage

The `s3-migrate` tool can be run from the command line with various flags to configure the migration process. Below is an example of how to use the tool:

```sh
./s3-migrate --source-key <SOURCE_ACCESS_KEY> --source-secret <SOURCE_SECRET> --source-region <SOURCE_REGION> --source-bucket <SOURCE_BUCKET> --source-endpoint <SOURCE_ENDPOINT> --dest-key <DEST_ACCESS_KEY> --dest-secret <DEST_SECRET> --dest-region <DEST_REGION> --dest-bucket <DEST_BUCKET> --dest-endpoint <DEST_ENDPOINT> --database <DATABASE_NAME> --collection <COLLECTION_NAME> --connection <MONGO_CONNECTION_URL> --filter '{"sizeint":{"$gt": 0}}' --limit 100 --ratelimit 10 --concurrency 5 --dry-run
```

Alternatively, you can use a configuration file named `s3-migrate.yaml` in the current folder to specify the configuration instead of using flags. The tool will automatically detect and use this file if present.

### Example `s3-migrate.yaml`

```yaml
# Source S3 Configuration
source-key: "your_source_access_key"
source-secret: "your_source_secret_key"
source-region: "us-east-1"
source-bucket: "source-bucket-name"
source-endpoint: "s3.amazonaws.com"  # Optional, defaults to s3.amazonaws.com

# Destination S3 Configuration
dest-key: "your_dest_access_key"
dest-secret: "your_dest_secret_key"
dest-region: "us-west-2"
dest-bucket: "destination-bucket-name"
dest-endpoint: "s3.amazonaws.com"  # Optional, defaults to s3.amazonaws.com

# MongoDB Configuration
database: "your_database_name"
collection: "your_collection_name"
connection: "mongodb://username:password@localhost:27017"
filter: '{"sizeint":{"$gt": 0}}'  # Example filter for objects with size > 0

# Performance Settings
limit: 100  # Number of documents to process per batch
ratelimit: 10  # Rate limit for S3 operations per second

# Optional Settings
cpuprofile: "profile.cpu"  # Optional, for CPU profiling
```

### Running with Docker

You can also run the `s3-migrate` tool using Docker. Pull the Docker image from GitHub Container Registry and run it with the necessary flags:

```sh
docker pull ghcr.io/highercomve/s3-migrate:main
docker run --rm ghcr.io/highercomve/s3-migrate:main \
  --source-key <SOURCE_ACCESS_KEY> --source-secret <SOURCE_SECRET> --source-region <SOURCE_REGION> --source-bucket <SOURCE_BUCKET> --source-endpoint <SOURCE_ENDPOINT> \
  --dest-key <DEST_ACCESS_KEY> --dest-secret <DEST_SECRET> --dest-region <DEST_REGION> --dest-bucket <DEST_BUCKET> --dest-endpoint <DEST_ENDPOINT> \
  --database <DATABASE_NAME> --collection <COLLECTION_NAME> --connection <MONGO_CONNECTION_URL> --filter '{"sizeint":{"$gt": 0}}' --limit 100 --ratelimit 10 --concurrency 5 --dry-run
```

Alternatively, you can use a configuration file named `s3-migrate.yaml` and mount it into the Docker container:

```sh
docker run --rm -v $(pwd)/s3-migrate.yaml:/app/s3-migrate.yaml ghcr.io/highercomve/s3-migrate:main
```

### Flags

- **Config and Profiling Flags**:
  - `--config`: Path to the config file (default is `$HOME/.cobra.yaml`).
  - `--cpuprofile`: Path to the file for CPU profiling.

- **Source Bucket Flags**:
  - `--source-key`: Source S3 access key.
  - `--source-secret`: Source S3 secret key.
  - `--source-region`: Source S3 region.
  - `--source-bucket`: Source S3 bucket name.
  - `--source-endpoint`: Source S3 endpoint.

- **Destination Bucket Flags**:
  - `--dest-key`: Destination S3 access key.
  - `--dest-secret`: Destination S3 secret key.
  - `--dest-region`: Destination S3 region.
  - `--dest-bucket`: Destination S3 bucket name.
  - `--dest-endpoint`: Destination S3 endpoint.

- **Database Flags**:
  - `--database`: MongoDB database name.
  - `--collection`: MongoDB collection name.
  - `--connection`: MongoDB connection URL.
  - `--filter`: MongoDB filter to select records for migration.

- **Performance Flags**:
  - `--limit`: Batch size for fetching records from MongoDB.
  - `--ratelimit`: Rate limit for S3 requests (operations per second).
  - `--concurrency`: Concurrency level for the migration process.
  - `--dry-run`: Enable dry run mode to simulate the migration without actual data transfer.

## Example

To migrate objects from a source S3 bucket to a destination S3 bucket based on MongoDB records, you can use the following command:

```sh
./s3-migrate --source-key "source-access-key" --source-secret "source-secret" --source-region "us-west-1" --source-bucket "source-bucket" --source-endpoint "https://s3.amazonaws.com" --dest-key "dest-access-key" --dest-secret "dest-secret" --dest-region "us-west-2" --dest-bucket "dest-bucket" --dest-endpoint "https://s3.amazonaws.com" --database "mydatabase" --collection "mycollection" --connection "mongodb://localhost:27017" --filter '{"sizeint":{"$gt": 0}}' --limit 100 --ratelimit 10 --concurrency 5 --dry-run
```

This command will simulate the migration process, showing which objects would be migrated without actually performing the migration.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Author

Sergio Marin - [@highercomve](https://github.com/highercomve)
