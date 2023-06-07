package minio

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/structs"
	"github.com/minio/minio-go/v7"
)

type UploadParams struct {
	Source            string
	Destination       string
	Recursive         bool
	RemoveSourceFiles bool
	Md5sum            bool
	StopOnError       bool
	NotifyErrors      bool
	DisableMultipart  bool
}

func Upload(logger Logger, notifier Notifier, serverParams Params, uploadParams UploadParams) {
	uploadParams.Source = strings.TrimSuffix(uploadParams.Source, "/")
	sourceFile, err := os.Open(uploadParams.Source)
	if err != nil {
		logger.FatalWithFields(map[string]interface{}{
			"source": uploadParams.Source,
		},
			"Unable to open source path",
		)
		notify(notifier, logger, uploadParams, "Kaynak dosya/dizin açılamadı: "+uploadParams.Source)
	}
	sourceAbs, err := filepath.Abs(sourceFile.Name())
	sourceBase := filepath.Base(sourceAbs)
	if err != nil {
		logger.FatalWithFields(map[string]interface{}{
			"source": uploadParams.Source,
		},
			"Unable to get absolute path of the source",
		)
		notify(notifier, logger, uploadParams, "Kaynak dosya/dizinin tam yolu belirlenemedi: "+uploadParams.Source)
	}
	sourceFileInfo, err := sourceFile.Stat()
	if err != nil {
		logger.FatalWithFields(map[string]interface{}{
			"source": uploadParams.Source,
		},
			"Unable to stat source path",
		)
		notify(notifier, logger, uploadParams, "Kaynak dosya/dizin bilgileri alınamadı: "+uploadParams.Source)
	}
	sourceIsDir := sourceFileInfo.IsDir()
	logger.DebugWithFields(map[string]interface{}{
		"sourceIsDir": sourceIsDir,
	})

	uploadParams.Destination = strings.TrimSuffix(uploadParams.Destination, "/") + "/"
	bucket := strings.Split(uploadParams.Destination, "/")[0]
	objectNamePrefix := strings.TrimPrefix(uploadParams.Destination, bucket+"/")
	if objectNamePrefix != "" {
		objectNamePrefix = strings.TrimSuffix(objectNamePrefix, "/") + "/"
	}

	sourcePrefix := strings.TrimSuffix(sourceAbs, sourceBase)

	logger.DebugWithFields(map[string]interface{}{
		"source":           uploadParams.Source,
		"sourceAbs":        sourceAbs,
		"sourceBase":       sourceBase,
		"sourcePrefix":     sourcePrefix,
		"sourceIsDir":      sourceIsDir,
		"destination":      uploadParams.Destination,
		"bucket":           bucket,
		"objectNamePrefix": objectNamePrefix,
	})

	c, err := NewClient(&serverParams)
	if err != nil {
		logger.Fatal(err)
		notify(notifier, logger, uploadParams, "MinIO client oluşturulamadı: "+err.Error())
	}
	logger.DebugWithFields(map[string]interface{}{
		"client": c.Client,
	},
		"MinIO client initialized",
	)

	sourceFiles := make([]string, 0)
	if sourceIsDir {
		if !uploadParams.Recursive {
			logger.Fatal(errors.New("recursive flag must be used to upload directories"))
			notify(notifier, logger, uploadParams, "\"recursive\" parametresi kullanılmadığı için dizin yüklenemedi.")
		}
		sourceFiles, err = getFiles(sourceAbs)
		if err != nil {
			logger.Fatal(err)
			notify(notifier, logger, uploadParams, sourceAbs+" dizinindeki dosyalar alınamadı.")
		}
	} else {
		sourceFiles = append(sourceFiles, sourceAbs)
	}
	logger.Info(strconv.Itoa(len(sourceFiles)) + " files will be uploaded")

	uploaded := make([]string, 0)

	for _, file := range sourceFiles {
		objectName := objectNamePrefix + strings.TrimPrefix(file, sourcePrefix)

		logger.DebugWithFields(map[string]interface{}{
			"file":       file,
			"objectName": objectName,
		},
			"File will be uploaded",
		)

		uploadInfo, err := c.FPutObject(context.Background(), bucket, objectName, file, minio.PutObjectOptions{
			DisableMultipart: uploadParams.DisableMultipart,
		})

		if err != nil {
			errorOrFatal(logger, notifier, uploadParams,
				map[string]interface{}{
					"file":  file,
					"error": err,
				},
				"`"+file+"` konumundaki dosya MinIO'ya yüklenemedi. (Alınan hata: `"+err.Error()+"`) Yükleme işleminde `--stop-on-error` parametresi kullanıldığı için devam edilmeyecek.",
				"Unable to upload file")
			continue
		}
		logger.InfoWithFields(map[string]interface{}{
			"file": file,
		},
			"File uploaded",
		)
		logger.DebugWithFields(structs.Map(uploadInfo), "File uploaded")

		if uploadParams.Md5sum {
			md5sum, err := md5sum(file)
			if err != nil {
				errorOrFatal(logger, notifier, uploadParams,
					map[string]interface{}{
						"file":  file,
						"error": err,
					},
					"`"+file+"` konumundaki dosya MinIO'ya yüklendi, ancak MD5 hash kontrolü için lokal dosyanın hash'i alınamadı. Yükleme işleminde `--stop-on-error` parametresi kullanıldığı için devam edilmeyecek.",
					"Unable to get md5sum of the file")
				continue
			}

			if strings.Contains(uploadInfo.ETag, "-") {
				logger.WarnWithFields(map[string]interface{}{
					"md5sum": md5sum,
					"Etag":   uploadInfo.ETag,
				},
					"Etag is not a valid md5sum (probably because file uploaded with multipart method). No md5sum validation will be made for this file.",
				)
			} else if md5sum != uploadInfo.ETag {
				errorOrFatal(logger, notifier, uploadParams,

					map[string]interface{}{
						"md5sum": md5sum,
						"ETag":   uploadInfo.ETag,
					},
					"`"+file+"` konumundaki dosya MinIO'ya yüklendi, ancak yüklenen dosya ile lokal dosyanın MD5 hash'leri aynı değil. Yükleme işleminde `--stop-on-error` parametresi kullanıldığı için devam edilmeyecek.",
					"md5sums don't match")
				continue
			} else {
				logger.DebugWithFields(map[string]interface{}{
					"md5sum": md5sum,
					"ETag":   uploadInfo.ETag,
				},
					"md5sums match",
				)
			}

			if uploadParams.RemoveSourceFiles {
				err = os.Remove(file)
				if err != nil {
					logger.Error(err)
				} else {
					logger.InfoWithFields(map[string]interface{}{
						"file": file,
					},
						"Source file removed",
					)
				}
			}
		}
		uploaded = append(uploaded, file)
	}

	notUploaded := len(sourceFiles) - len(uploaded)

	if notUploaded > 0 {
		notify(notifier, logger, uploadParams, strconv.Itoa(notUploaded)+" dosya MinIO'ya yüklenemedi. Lütfen logu kontrol edin.")
	}

}

func md5sum(file string) (string, error) {
	var md5Str string

	f, err := os.Open(file)
	if err != nil {
		return md5Str, err
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	hash := md5.New()

	if _, err := io.Copy(hash, f); err != nil {
		return md5Str, err
	}

	hashInBytes := hash.Sum(nil)[:16]

	md5Str = hex.EncodeToString(hashInBytes)

	return md5Str, nil
}

func notify(notifier Notifier, logger Logger, params UploadParams, text string) {
	if params.NotifyErrors {
		err := notifier.Notify(text)
		if err != nil {
			logger.WarnWithFields(map[string]interface{}{
				"error": err,
			},
				"Unable to send notification",
			)
		}
	}
}

func errorOrFatal(logger Logger, notifier Notifier, params UploadParams, fields map[string]interface{}, notification string, args ...interface{}) {
	if params.StopOnError {
		notify(notifier, logger, params, notification)
		logger.FatalWithFields(fields, args)
	} else {
		logger.ErrorWithFields(fields, args)
	}
}

func getFiles(path string) ([]string, error) {
	Files := make([]string, 0)
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			Files = append(Files, filePath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return Files, nil
}
