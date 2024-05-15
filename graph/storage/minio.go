package storage

import (
	"context"
	"io"
	"log"
	"os"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var singleMinioInstance *minio.Client

var minioOnce sync.Once

func GetMinioInstance() *minio.Client {
	minioOnce.Do(func() {
		client, err := minio.New(os.Getenv("MINIO_ENDPOINT"), &minio.Options{
			Creds:  credentials.NewStaticV4(os.Getenv("MINIO_ACCESS_KEY"), os.Getenv("MINIO_ACCESS_SECRET"), ""),
			Secure: false,
		})
		if err != nil {
			log.Fatalf("unable to connect to minio: %v\n", err)
		}
		bucketName := os.Getenv("MINIO_BUCKET_NAME")
		err = client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
		if err != nil {
			exists, errBucketExists := client.BucketExists(context.Background(), bucketName)
			if errBucketExists == nil && exists {
				log.Printf("bucket %s already exists\n", bucketName)
			} else {
				log.Fatalf("unable to create bucket: %v\n", err)
			}
		} else {
			log.Printf("successfully created bucket %s\n", bucketName)
		}
		singleMinioInstance = client
	})
	return singleMinioInstance
}

func Upload(ctx context.Context, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	minioClient := GetMinioInstance()
	return minioClient.PutObject(ctx, os.Getenv("MINIO_BUCKET_NAME"), objectName, reader, objectSize, opts)
}
