package main

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"go.mau.fi/whatsmeow/appstate"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// Handler is a simple handler for incoming events.
func handler(rawEvt interface{}) {
	switch evt := rawEvt.(type) {

	case *events.AppStateSyncComplete:
		if len(cli.Store.PushName) > 0 && evt.Name == appstate.WAPatchCriticalBlock {
			err := cli.SendPresence(types.PresenceAvailable)
			if err != nil {
				log.Warnf("Failed to send available presence: %v", err)
			} else {
				log.Infof("Marked self as available")
			}
		}
	case *events.Connected, *events.PushNameSetting:
		if len(cli.Store.PushName) == 0 {
			return
		}
		// Send presence available when connecting and when the pushname is changed.
		// This makes sure that outgoing messages always have the right pushname.
		err := cli.SendPresence(types.PresenceAvailable)
		if err != nil {
			log.Warnf("Failed to send available presence: %v", err)
		} else {
			log.Infof("Marked self as available")
		}
	case *events.StreamReplaced:
		os.Exit(0)
	case *events.Message:
		metaParts := []string{fmt.Sprintf("pushname: %s", evt.Info.PushName), fmt.Sprintf("timestamp: %s", evt.Info.Timestamp)}
		if evt.Info.Type != "" {
			metaParts = append(metaParts, fmt.Sprintf("type: %s", evt.Info.Type))
		}
		if evt.Info.Category != "" {
			metaParts = append(metaParts, fmt.Sprintf("category: %s", evt.Info.Category))
		}
		if evt.IsViewOnce {
			metaParts = append(metaParts, "view once")
		}
		if evt.IsViewOnce {
			metaParts = append(metaParts, "ephemeral")
		}
		if evt.IsViewOnceV2 {
			metaParts = append(metaParts, "ephemeral (v2)")
		}
		if evt.IsDocumentWithCaption {
			metaParts = append(metaParts, "document with caption")
		}
		if evt.IsEdit {
			metaParts = append(metaParts, "edit")
		}

		log.Infof("Received message %s from %s (%s): %+v", evt.Info.ID, evt.Info.SourceString(), strings.Join(metaParts, ", "), evt.Message)

		if evt.Message.GetPollUpdateMessage() != nil {
			decrypted, err := cli.DecryptPollVote(evt)
			if err != nil {
				log.Errorf("Failed to decrypt vote: %v", err)
			} else {
				log.Infof("Selected options in decrypted vote:")
				for _, option := range decrypted.SelectedOptions {
					log.Infof("- %X", option)
				}
			}
		} else if evt.Message.GetEncReactionMessage() != nil {
			decrypted, err := cli.DecryptReaction(evt)
			if err != nil {
				log.Errorf("Failed to decrypt encrypted reaction: %v", err)
			} else {
				log.Infof("Decrypted reaction: %+v", decrypted)
			}
		}

		img := evt.Message.GetImageMessage()
		if img != nil {
			data, err := cli.Download(img)
			if err != nil {
				log.Errorf("Failed to download image: %v", err)
				return
			}
			exts, _ := mime.ExtensionsByType(img.GetMimetype())
			path := fmt.Sprintf("%s%s", evt.Info.ID, exts[0])
			err = os.WriteFile(path, data, 0600)
			if err != nil {
				log.Errorf("Failed to save image: %v", err)
				return
			}
			log.Infof("Saved image in message to %s", path)
		}
		var msgContent string

		switch {
		case evt.Message.GetConversation() != "":
			msgContent = evt.Message.GetConversation()
		case evt.Message.GetExtendedTextMessage() != nil:
			msgContent = evt.Message.GetExtendedTextMessage().GetText()
		case evt.Message.GetImageMessage() != nil:
			msgContent = evt.Message.GetImageMessage().GetCaption()
		case evt.Message.GetDocumentMessage() != nil:
			msgContent = evt.Message.GetDocumentMessage().GetCaption()
		case evt.Message.GetVideoMessage() != nil:
			msgContent = evt.Message.GetVideoMessage().GetCaption()
		}

		remotejid := evt.Info.MessageSource.Chat.String()

		if err := insertMessageHistory(evt.Info.ID, cli.Store.ID.String(), remotejid, evt.Info.Type, msgContent, evt.Info.Timestamp, evt.Info.MessageSource.IsFromMe); err != nil {
			log.Errorf("Failed to insert message history: %v", err)
		}

		if err := insertLastMessages(evt.Info.ID, cli.Store.ID.String(), remotejid, evt.Info.Type, msgContent, evt.Info.Timestamp, evt.Info.MessageSource.IsFromMe); err != nil {
			log.Errorf("Failed to insert last messages: %v", err)
		}

	case *events.Receipt:
		if evt.Type == events.ReceiptTypeRead || evt.Type == events.ReceiptTypeReadSelf {
			log.Infof("%v was read by %s at %s", evt.MessageIDs, evt.SourceString(), evt.Timestamp)
		} else if evt.Type == events.ReceiptTypeDelivered {
			log.Infof("%s was delivered to %s at %s", evt.MessageIDs[0], evt.SourceString(), evt.Timestamp)
		}
	case *events.Presence:
		if evt.Unavailable {
			if evt.LastSeen.IsZero() {
				log.Infof("%s is now offline", evt.From)
			} else {
				log.Infof("%s is now offline (last seen: %s)", evt.From, evt.LastSeen)
			}
		} else {
			log.Infof("%s is now online", evt.From)
		}
	case *events.HistorySync:
		id := atomic.AddInt32(&historySyncID, 1)
		fileName := fmt.Sprintf("history-%d-%d.json", startupTime, id)
		file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			log.Errorf("Failed to open file to write history sync: %v", err)
			return
		}
		enc := json.NewEncoder(file)
		enc.SetIndent("", "  ")
		err = enc.Encode(evt.Data)
		if err != nil {
			log.Errorf("Failed to write history sync: %v", err)
			return
		}
		log.Infof("Wrote history sync to %s", fileName)
		_ = file.Close()
	case *events.AppState:
		log.Debugf("App state event: %+v / %+v", evt.Index, evt.SyncActionValue)
	case *events.KeepAliveTimeout:
		log.Debugf("Keepalive timeout event: %+v", evt)
		if evt.ErrorCount > 3 {
			log.Debugf("Got >3 keepalive timeouts, forcing reconnect")
			go func() {
				cli.Disconnect()
				err := cli.Connect()
				if err != nil {
					log.Errorf("Error force-reconnecting after keepalive timeouts: %v", err)
				}
			}()
		}
	case *events.KeepAliveRestored:
		log.Debugf("Keepalive restored")
	}
}

func handleCmd(cmd string, args []string) {
	switch cmd {
	case "isloggedin":
		handleIsLoggedIn()
	case "checkuser":
		handleCheckUser(args)
	case "send":
		handleSendMessage(args)
	}
}

func handleIsLoggedIn() {
	log.Infof("Checking if logged in...")
	log.Infof("Logged in: %t", cli.IsLoggedIn())
}

func handleCheckUser(args []string) {
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

func handleSendMessage(args []string) {
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

	if err := insertMessageHistory(resp.ID, cli.Store.ID.String(), recipient.String(), "text", msg.GetConversation(), resp.Timestamp, true); err != nil {
		log.Errorf("Error inserting message history: %v", err)
	}

	if err := insertLastMessages(resp.ID, cli.Store.ID.String(), recipient.String(), "text", msg.GetConversation(), resp.Timestamp, true); err != nil {
		log.Errorf("Error inserting last messages: %v", err)
	}
}

// Parse a JID from a string. If the string starts with a +, it is removed.
func parseJID(arg string) (types.JID, bool) {
	if arg[0] == '+' {
		arg = arg[1:]
	}
	if !strings.ContainsRune(arg, '@') {
		return types.NewJID(arg, types.DefaultUserServer), true
	} else {
		recipient, err := types.ParseJID(arg)
		if err != nil {
			log.Errorf("Invalid JID %s: %v", arg, err)
			return recipient, false
		} else if recipient.User == "" {
			log.Errorf("Invalid JID %s: no server specified", arg)
			return recipient, false
		}
		return recipient, true
	}
}

// insertMessageHistory inserts a message history record into the database.
//
// The message history includes details such as message ID, device JID, remote JID,
// message content, message type, timestamp, and sent status.
//
// Parameters:
//   - messageID: The ID of the message.
//   - device_jid: The JID of the device (current session owner, our JID).
//   - remote_jid: The JID of the remote user.
//   - messageContent: The content of the message.
//   - messageType: The type of the message.
//   - timestamp: The timestamp when the message was sent.
//   - sent: A boolean indicating if the message was sent.
//
// Returns:
//   - error: An error if the insertion fails, or nil if successful.
func insertMessageHistory(messageID, device_jid, remote_jid, messageContent, messageType string, timestamp time.Time, sent bool) error {
	_, err := db.Exec(`
		INSERT INTO message_history (message_id, device_jid, remote_jid, message_type, message_content, timestamp, sent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, messageID, device_jid, remote_jid, messageContent, messageType, timestamp, sent)
	if err != nil {
		return fmt.Errorf("failed to insert message history: %w", err)
	}
	log.Infof("Inserted message history: %d, %s, %s, %s, %s", messageID, device_jid, remote_jid, messageContent, timestamp)
	return nil
}

// insertLastMessages inserts or updates the last message for a remote JID in the database.
//
// If the remote JID is not present in the database, it inserts a new record.
// If the remote JID is already present, it updates the existing record with the new message details.
//
// Parameters:
//   - messageID: The ID of the message.
//   - device_jid: The JID of the device.
//   - remote_jid: The JID of the remote user.
//   - messageContent: The content of the message.
//   - messageType: The type of the message.
//   - timestamp: The timestamp when the message was sent.
//   - sent: A boolean indicating if the message was sent.
//
// Returns:
//   - error: An error if the insertion/update fails, or nil if successful.
func insertLastMessages(messageID, device_jid, remote_jid, messageContent, messageType string, timestamp time.Time, sent bool) error {
	_, err := db.Exec(`
		INSERT INTO last_messages (message_id, device_jid, remote_jid, message_type, message_content, timestamp, sent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (remote_jid)
		DO UPDATE SET message_id = $1, device_jid = $2, message_type = $4, message_content = $5, timestamp = $6, sent = $7
	`, messageID, device_jid, remote_jid, messageContent, messageType, timestamp, sent)
	if err != nil {
		return fmt.Errorf("failed to insert last messages: %w", err)
	}
	log.Infof("Inserted last messages: %d, %s, %s, %s, %s", messageID, device_jid, remote_jid, messageContent, timestamp)
	return nil
}
