package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	_ "github.com/lib/pq"
)

func main() {
	// Provide your AWS credentials
	accessKey := os.Getenv("YOUR_AWS_ACCESS_KEY")
	secretKey := os.Getenv("YOUR_AWS_SECRET_KEY")

	bucketName := os.Getenv("YOUR_BUCKET_NAME")
	endpointURL := os.Getenv("S3ENDURL")
	regionS3 := os.Getenv("REGION")

	// POST request handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check the request method
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Reading the request body
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		// Checking the value of the request body
		if string(body) != `{"START"}` {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Generating a path to objects in an S3 bucket
		objectPrefix := os.Getenv("PREFIX")

		// start function
		err = DownloadMXFFilesFromS3(bucketName, objectPrefix, regionS3, accessKey, secretKey, endpointURL)
		if err != nil {
			log.Fatal(err)
		}

		// Sending a successful response
		w.WriteHeader(http.StatusOK)
	})

	// Running a web server on port 8080
	fmt.Println("Server listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Function downloads MXF files from the S3 bucket
func DownloadMXFFilesFromS3(bucketName, objectPrefix, region, accessKeyID, secretAccessKey, endpointURL string) error {
	// Create an AWS session with the specified credentials and region
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Endpoint:    aws.String(endpointURL),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return fmt.Errorf("failed to create AWS session: %v", err)
	}

	// Create a new S3 client
	svc := s3.New(sess)

	// Create a new pool of S3 loaders
	downloader := s3manager.NewDownloaderWithClient(svc, func(d *s3manager.Downloader) {
		d.Concurrency = 5 // Set the desired download concurrency value
	})

	// Specify the directory path to save files
	outputDirectory := "shared/"

	// Function to process each page of objects
	pageHandler := func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		// Iterate over each object and download files with .mxf extension
		for _, obj := range page.Contents {
			key := *obj.Key
			if filepath.Ext(key) == ".mxf" {
				// Form the path and file name to save
				filePath := filepath.Join(outputDirectory, filepath.Base(key))

				// Get the filename without extension
				filename := filepath.Base(key[:len(key)-len(filepath.Ext(key))])

				// Check if a value exists in the database
				if existsInDatabase(filename) {
					fmt.Printf("Skipping file (already exists): %s\n", filePath)
					continue // Skip the file and move to the next iteration of the loop
				}

				// Create a file for writing
				file, err := os.Create(filePath)
				if err != nil {
					log.Printf("failed to create file: %v", err)
				}
				defer file.Close()

				// Download the object from S3
				_, err = downloader.Download(file, &s3.GetObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(key),
				})
				if err != nil {
					fmt.Errorf("failed to download object from S3: %v", err)
				}

				// Here, the filename without extension is written to the database
				err = writeToDatabase(filepath.Base(key[:len(key)-len(filepath.Ext(key))]))
				if err != nil {
					fmt.Errorf("failed to write to database: %v", err)
				}

				fmt.Printf("Downloaded file: %s\n", filePath)
			}
		}

		// Return true to continue processing the next page
		return true
	}

	// Get the list of objects in S3 using pagination
	err = svc.ListObjectsV2Pages(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(objectPrefix),
	}, pageHandler)

	if err != nil {
		return fmt.Errorf("failed to list objects in S3: %v", err)
	}

	return nil
}

func writeToDatabase(filename string) error {
	// Connect to the PostgreSQL database
	dbConnectionString := os.Getenv("DB_CONNECTION_STRING")
	if dbConnectionString == "" {
		fmt.Println("DB_CONNECTION_STRING environment variable is not set")
	}

	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		log.Printf("Failed to connect to the database: %v", err)

	}
	defer db.Close()

	// Insert data into the table
	_, err = db.Exec("INSERT INTO transcode_jobs (id) VALUES ($1)", filename)
	if err != nil {
		log.Printf("Failed to insert data into the table: %v", err)
	}

	log.Printf("Data saved successfully: ID=%s", filename)

	return nil
}

func getEnvVariable(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatal(name + " environment variable is not set")
	}
	return value
}

func existsInDatabase(filename string) bool {
	// Connect to the PostgreSQL database
	dbConnectionString := os.Getenv("DB_CONNECTION_STRING")
	if dbConnectionString == "" {
		fmt.Println("DB_CONNECTION_STRING environment variable is not set")
	}

	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		log.Printf("Failed to connect to the database: %v", err)
		return false
	}
	defer db.Close()

	// Check the presence of the value in the table
	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM transcode_jobs WHERE id = $1", filename)
	if err := row.Scan(&count); err != nil {
		log.Printf("Failed to check value in the table: %v", err)
		return false
	}

	return count > 0
}
