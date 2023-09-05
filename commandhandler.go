package main

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"
)

func handleIsLoggedIn() {
	log.Infof("Checking if logged in...")
	log.Infof("Logged in: %t", cli.IsLoggedIn())
}

func handleCheckUser(args []string) {
	log.Infof("Checking users: %v", args)
	if len(args) < 1 {
		log.Errorf("Usage: checkuser <phone numbers...>")
		return
	}

	resp, err := cli.IsOnWhatsApp(args)
	if err != nil {
		log.Errorf("Failed to check if users are on WhatsApp: %v", err)
		return
	}

	for _, item := range resp {
		logMessage := fmt.Sprintf("%s: on WhatsApp: %t, JID: %s", item.Query, item.IsIn, item.JID)

		if item.VerifiedName != nil {
			logMessage += fmt.Sprintf(", business name: %s", item.VerifiedName.Details.GetVerifiedName())
		}

		log.Infof(logMessage)

		// Send response to websocket
		if wsConn != nil {
			wsConn.WriteJSON(item)
		}
	}
}

func handleSendTextMessage(args []string, userID int) {
	if len(args) < 2 {
		log.Errorf("Usage: send <jid> <text>")
		return
	}

	recipient, ok := parseJID(args[0])
	if !ok {
		return
	}

	msg := &waProto.Message{
		Conversation: proto.String(strings.Join(args[1:], " ")),
	}
	log.Infof("Sending message to %s: %s", recipient, msg.GetConversation())

	resp, err := cli.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		log.Errorf("Error sending message: %v", err)
		return
	}

	log.Infof("Message sent (server timestamp: %s)", resp.Timestamp)

	if err := insertMessages(resp.ID, cli.Store.ID.String(), recipient.String(), msg.GetConversation(), "text", resp.Timestamp, true, "", userID); err != nil {
		log.Errorf("Error inserting into messages: %v", err)
	}

	if err := insertLastMessages(resp.ID, cli.Store.ID.String(), recipient.String(), msg.GetConversation(), "text", resp.Timestamp, true, "", userID); err != nil {
		log.Errorf("Error inserting into last_messages: %v", err)
	}

	if wsConn != nil {
		m := Message{resp.ID, recipient.String(), "text", msg.GetConversation(), true, ""}
		wsConn.WriteJSON(m)
	}
}

func handleMarkRead(args []string) {
	if len(args) < 2 {
		log.Errorf("Usage: markread <message_id> <remote_jid>")
		return
	}

	messageID := args[0]
	remoteJID := args[1]

	if remoteJID == "" {
		log.Errorf("Invalid remote JID")
		return
	}

	sender, ok := parseJID(remoteJID)
	if !ok {
		return
	}

	timestamp := time.Now()

	if err := cli.MarkRead([]string{messageID}, timestamp, sender, sender); err != nil {
		log.Errorf("Error marking read: %v", err)
		return
	}
	log.Infof("MarkRead sent: %s %s %s", messageID, timestamp, sender)

	if err := markMessageRead(messageID, remoteJID, timestamp); err != nil {
		log.Errorf("Error marking message as read: %v", err)
	}
}

func handleSendImage(JID string, userID int, data []byte) error {
	recipient, ok := parseJID(JID)
	if !ok {
		return fmt.Errorf("invalid JID")
	}

	uploaded, err := cli.Upload(context.Background(), data, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("failed to upload file: %v", err)
	}

	msg := createImageMessage(uploaded, &data)
	resp, err := cli.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		return fmt.Errorf("error sending image message: %v", err)
	}

	log.Infof("Image message sent (server timestamp: %s)", resp.Timestamp)

	if err := insertMessages(resp.ID, cli.Store.ID.String(), recipient.String(), "", "media", resp.Timestamp, true, "", userID); err != nil {
		return fmt.Errorf("error inserting into messages: %v", err)
	}

	if err := insertLastMessages(resp.ID, cli.Store.ID.String(), recipient.String(), "", "media", resp.Timestamp, true, "", userID); err != nil {
		return fmt.Errorf("error inserting into last_messages: %v", err)
	}

	saveImage(msg, data, resp.ID)

	if wsConn != nil {
		m := Message{resp.ID, recipient.String(), "media", "", true, ""}
		wsConn.WriteJSON(m)
	}

	return nil
}

func handleSendDocument(JID string, fileName string, userID int, data []byte) error {
	recipient, ok := parseJID(JID)
	if !ok {
		return fmt.Errorf("invalid JID")
	}

	uploaded, err := cli.Upload(context.Background(), data, whatsmeow.MediaDocument)
	if err != nil {
		return fmt.Errorf("failed to upload file: %v", err)
	}

	msg := createDocumentMessage(fileName, uploaded, &data)
	resp, err := cli.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		return fmt.Errorf("error sending document message: %v", err)
	}

	log.Infof("Document message sent (server timestamp: %s)", resp.Timestamp)

	if err := insertMessages(resp.ID, cli.Store.ID.String(), recipient.String(), "", "media", resp.Timestamp, true, fileName, userID); err != nil {
		return fmt.Errorf("error inserting into messages: %v", err)
	}

	if err := insertLastMessages(resp.ID, cli.Store.ID.String(), recipient.String(), "", "media", resp.Timestamp, true, fileName, userID); err != nil {
		return fmt.Errorf("error inserting into last_messages: %v", err)
	}

	saveDocument(msg, data, resp.ID)

	if wsConn != nil {
		m := Message{resp.ID, recipient.String(), "media", "", true, fileName}
		wsConn.WriteJSON(m)
	}

	return nil
}

func saveImage(msg *waProto.Message, data []byte, ID string) {
	exts, err := mime.ExtensionsByType(msg.GetImageMessage().GetMimetype())
	if err != nil {
		log.Errorf("Error getting file extension: %v", err)
		return
	}

	if len(exts) == 0 {
		log.Errorf("No file extension found for mimetype: %s", msg.GetImageMessage().GetMimetype())
		return
	}

	extension := exts[0]
	fileName := fmt.Sprintf("%s%s", ID, extension)

	if err := uploadFile(*minioBucket, fileName, data); err != nil {
		log.Errorf("Error uploading image: %v", err)
		return
	}

	img, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		log.Errorf("Error decoding image: %v", err)
		return
	}

	thumbnail := imaging.Thumbnail(img, 100, 100, imaging.Lanczos)

	thumbnailData := new(bytes.Buffer)
	if err := imaging.Encode(thumbnailData, thumbnail, imaging.JPEG); err != nil {
		log.Errorf("Error encoding thumbnail: %v", err)
		return
	}

	thumbnailFileName := fmt.Sprintf("%s%s", ID, ".jpg")
	if err := uploadFile(*minioBucket, thumbnailFileName, thumbnailData.Bytes()); err != nil {
		log.Errorf("Error uploading thumbnail: %v", err)
		return
	}

	log.Infof("Uploaded image to %s", fileName)
	log.Infof("Uploaded thumbnail to %s", thumbnailFileName)
}

func saveDocument(msg *waProto.Message, data []byte, ID string) {
	exts, err := mime.ExtensionsByType(msg.GetDocumentMessage().GetMimetype())
	if err != nil {
		log.Errorf("Error getting file extension: %v", err)
		return
	}

	if len(exts) == 0 {
		log.Errorf("No file extension found for mimetype: %s", msg.GetDocumentMessage().GetMimetype())
		return
	}

	extension := exts[0]
	path := fmt.Sprintf("%s%s", ID, extension)

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		log.Errorf("Error saving file to disk: %v", err)
		return
	}

	log.Infof("Saved file to %s", path)
}

func createImageMessage(uploaded whatsmeow.UploadResponse, data *[]byte) *waProto.Message {
	return &waProto.Message{
		ImageMessage: &waProto.ImageMessage{
			Url:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(http.DetectContentType(*data)),
			FileEncSha256: uploaded.FileEncSHA256,
			FileSha256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(*data))),
		},
	}
}

func createDocumentMessage(fileName string, uploaded whatsmeow.UploadResponse, data *[]byte) *waProto.Message {
	return &waProto.Message{
		DocumentMessage: &waProto.DocumentMessage{
			FileName:      proto.String(fileName),
			Url:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(http.DetectContentType(*data)),
			FileEncSha256: uploaded.FileEncSHA256,
			FileSha256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(*data))),
			Title:         proto.String(fmt.Sprintf("%s%s", "document", filepath.Ext(uploaded.URL))),
		},
	}
}
