package minio

import (
	"crypto/tls"
	"net/http"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	*minio.Client
	params *Params
}

type Params struct {
	Endpoint           string
	AccessKey          string
	SecretKey          string
	Secure             bool
	InsecureSkipVerify bool
}

func NewClient(params *Params) (*Client, error) {
	minioOptions := &minio.Options{
		Creds:  credentials.NewStaticV4(params.AccessKey, params.SecretKey, ""),
		Secure: params.Secure,
	}
	if params.InsecureSkipVerify {
		customTransport := http.DefaultTransport.(*http.Transport).Clone()
		customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		minioOptions.Transport = customTransport
	}
	minioClient, err := minio.New(params.Endpoint, minioOptions)
	if err != nil {
		return nil, err
	}

	c := Client{minioClient, params}

	return &c, nil
}
