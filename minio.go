package main

import (
	"whatsapp-ws/minio"

	waProto "go.mau.fi/whatsmeow/binary/proto"
)

func saveToMinio(img *waProto.ImageMessage) (string, error) {
	minio.Upload(img)
}
