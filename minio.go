package main

import (
	"bytes"
	"context"

	"github.com/minio/minio-go/v7"
)

func uploadFile(bucket, objectName string, data []byte) error {
	buf := bytes.NewBuffer(data)
	_, err := minioClient.PutObject(context.Background(), bucket, objectName, buf, int64(len(data)), minio.PutObjectOptions{ContentType: "application/octet-stream"})

	if err != nil {
		return err
	}

	return nil
}
