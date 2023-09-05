package main

import (
	"bytes"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
)

func uploadFile(bucket, objectName string, data []byte) error {
	buf := bytes.NewBuffer(data)
	_, err := minioClient.PutObject(context.Background(), bucket, objectName, buf, int64(len(data)), minio.PutObjectOptions{ContentType: "application/octet-stream"})

	if err != nil {
		switch errResponse := minio.ToErrorResponse(err); errResponse.Code {
		case "NoSuchBucket":
			return fmt.Errorf("bucket does not exist")
		case "AccessDenied":
			return fmt.Errorf("access denied")
		default:
			return err
		}
	}

	return nil
}
