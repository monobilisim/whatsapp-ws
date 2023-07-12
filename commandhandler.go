package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

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

	if err := insertMessages(resp.ID, cli.Store.ID.String(), recipient.String(), "text", msg.GetConversation(), resp.Timestamp, true, "", "", userID); err != nil {
		log.Errorf("Error inserting into messages: %v", err)
	}

	if err := insertLastMessages(resp.ID, cli.Store.ID.String(), recipient.String(), "text", msg.GetConversation(), resp.Timestamp, true, "", "", userID); err != nil {
		log.Errorf("Error inserting into last_messages: %v", err)
	}

	if wsConn != nil {
		m := Message{resp.ID, recipient.String(), msg.GetConversation(), true}
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

	sender, ok := parseJID(remoteJID)
	if !ok {
		return
	}

	timestamp := time.Now()

	err := cli.MarkRead([]string{messageID}, timestamp, sender, sender)
	if err != nil {
		log.Errorf("Error marking read: %v", err)
		return
	}
	log.Infof("MarkRead sent: %s %s %s", messageID, timestamp, sender)

	if err := markMessageRead(messageID, remoteJID, timestamp); err != nil {
		log.Errorf("Error marking message as read: %v", err)
	}
}

func handleSendImage(JID string, caption string, userID int, data []byte) {

	recipient, ok := parseJID(JID)
	if !ok {
		return
	}

	uploaded, err := cli.Upload(context.Background(), data, whatsmeow.MediaImage)
	if err != nil {
		log.Errorf("Failed to upload file: %v", err)
		return
	}
	msg := &waProto.Message{ImageMessage: &waProto.ImageMessage{
		Caption:       proto.String(caption),
		Url:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(http.DetectContentType(data)),
		FileEncSha256: uploaded.FileEncSHA256,
		FileSha256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
	}}
	resp, err := cli.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		log.Errorf("Error sending image message: %v", err)
	} else {
		log.Infof("Image message sent (server timestamp: %s)", resp.Timestamp)
	}
}

func handleSendDocument(JID string, caption string, userID int, data []byte) {

	recipient, ok := parseJID(JID)
	if !ok {
		return
	}
	uploaded, err := cli.Upload(context.Background(), data, whatsmeow.MediaDocument)
	if err != nil {
		log.Errorf("Failed to upload file: %v", err)
		return
	}
	msg := &waProto.Message{DocumentMessage: &waProto.DocumentMessage{
		Caption:       proto.String(caption),
		Url:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(http.DetectContentType(data)),
		FileEncSha256: uploaded.FileEncSHA256,
		FileSha256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(data))),
		Title:         proto.String("test"),
	}}
	resp, err := cli.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		log.Errorf("Error sending document message: %v", err)
	} else {
		log.Infof("Document message sent (server timestamp: %s)", resp.Timestamp)
	}
}
