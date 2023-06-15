package main

import (
	"context"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// WIP
func UploadFile(endpoint, accessKey, secretKey, bucket, objectName, filePath string) error {
	// Initialize MinIO client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: true,
	})
	if err != nil {
		return err
	}

	// Upload the file with FPutObject
	_, err = minioClient.FPutObject(context.Background(), bucket, objectName, filePath, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return err
	}

	log.Infof("Successfully uploaded %s to %s\n", filePath, objectName)
	return nil
}
