package main

import (
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"strings"
	"sync/atomic"

	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

func handleAppStateSyncComplete(evt *events.AppStateSyncComplete) {
	if len(cli.Store.PushName) > 0 && evt.Name == appstate.WAPatchCriticalBlock {
		err := cli.SendPresence(types.PresenceAvailable)
		if err != nil {
			log.Warnf("Failed to send available presence: %v", err)
		} else {
			log.Infof("Marked self as available")
		}
	}
}

func handleConnectedOrPushNameSetting(evt interface{}) {
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
}

func handleStreamReplaced(evt *events.StreamReplaced) {
	os.Exit(0)
}

func handleMessage(evt *events.Message) {
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

	if evt.Message.GetProtocolMessage() != nil {
		return
	}

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

	var extension string
	var fileName string

	if img := evt.Message.GetImageMessage(); img != nil {
		data, err := cli.Download(img)
		if err != nil {
			log.Errorf("Failed to download image: %v", err)
			return
		}

		fileName = fmt.Sprintf("%s%s", evt.Info.ID, getFileExtension(img.GetMimetype()))
		if err := uploadFile(*minioBucket, fileName, data); err != nil {
			log.Errorf("Failed to upload image: %v", err)
			return
		}

		thumb := img.GetJpegThumbnail()
		if err := uploadFile(*minioBucket, fmt.Sprintf("%s%s", evt.Info.ID, ".jpg"), thumb); err != nil {
			log.Errorf("Failed to upload thumbnail: %v", err)
			return
		}

		log.Infof("Uploaded image message to %s", fileName)
	}

	if doc := evt.Message.GetDocumentMessage(); doc != nil {
		data, err := cli.Download(doc)
		if err != nil {
			log.Errorf("Failed to download document: %v", err)
			return
		}
		exts, _ := mime.ExtensionsByType(doc.GetMimetype())
		extension = exts[0]
		path := fmt.Sprintf("%s%s", evt.Info.ID, extension)
		err = os.WriteFile(path, data, 0644)
		if err != nil {
			log.Errorf("Failed to save document: %v", err)
			return
		}

		fileName = doc.GetFileName()

		if err := uploadFile(*minioBucket, fileName, data); err != nil {
			log.Errorf("Failed to upload document: %v", err)
			return
		}

		thumb := doc.GetJpegThumbnail()
		if thumb != nil {
			if err := uploadFile(*minioBucket, fmt.Sprintf("%s%s", fileName, ".jpg"), thumb); err != nil {
				log.Errorf("Failed to upload document thumbnail: %v", err)
				return
			}
		}

		log.Infof("Uploaded document message: %s", fileName)
	}

	if audio := evt.Message.GetAudioMessage(); audio != nil {
		data, err := cli.Download(audio)
		if err != nil {
			log.Errorf("Failed to download audio: %v", err)
			return
		}

		fileName = fmt.Sprintf("%s%s", evt.Info.ID, getFileExtension(audio.GetMimetype()))

		if err := uploadFile(*minioBucket, fileName, data); err != nil {
			log.Errorf("Failed to upload audio: %v", err)
			return
		}

		log.Infof("Uploaded audio message: %s", fileName)
	}

	var msgContent string
	var msgType string

	switch {
	case evt.Message.GetConversation() != "":
		msgContent = evt.Message.GetConversation()
		msgType = "text"
	case evt.Message.GetExtendedTextMessage() != nil:
		msgContent = evt.Message.GetExtendedTextMessage().GetText()
		msgType = "text"
	case evt.Message.GetImageMessage() != nil:
		msgContent = evt.Message.GetImageMessage().GetCaption()
		msgType = "media"
	case evt.Message.GetDocumentMessage() != nil:
		msgContent = evt.Message.GetDocumentMessage().GetCaption()
		msgType = "media"
	case evt.Message.GetVideoMessage() != nil:
		msgContent = evt.Message.GetVideoMessage().GetCaption()
		msgType = "media"
	}

	remoteJid := evt.Info.MessageSource.Chat.String()

	if evt.Info.Category == "peer" {
		// Bunlar ilk login olunduğunda alınan sistem mesajları, veritabanına yazmayalım.
		// Örn: [Main INFO] Received message xxxxxxxxxxxxxxxxxxxxxxxxxxxxx from xxxxxxxxx@s.whatsapp.net (pushname: xxxxxx, timestamp: 2023-06-21 12:16:33 +0300 +03, type: text, category: peer): protocolMessage:{type:INITIAL_SECURITY_NOTIFICATION_SETTING_SYNC initialSecurityNotificationSettingSync:{securityNotificationEnabled:false}}
		return
	}

	if remoteJid == "status@broadcast" {
		return
	}

	if err := insertMessages(evt.Info.ID, cli.Store.ID.String(), remoteJid, msgContent, msgType, evt.Info.Timestamp, evt.Info.MessageSource.IsFromMe, fileName, -1); err != nil {
		log.Errorf("Error inserting into messages: %v", err)
	}

	if err := insertLastMessages(evt.Info.ID, cli.Store.ID.String(), remoteJid, msgContent, msgType, evt.Info.Timestamp, evt.Info.MessageSource.IsFromMe, fileName, -1); err != nil {
		log.Errorf("Error inserting into last_messages: %v", err)
	}

	if wsConn != nil {
		m := Message{evt.Info.ID, remoteJid, msgType, msgContent, evt.Info.MessageSource.IsFromMe, fileName}
		wsConn.WriteJSON(m)
	}
}

func handleReceipt(evt *events.Receipt) {
	if evt.Type == events.ReceiptTypeRead || evt.Type == events.ReceiptTypeReadSelf {
		log.Infof("%v was read by %s at %s", evt.MessageIDs, evt.SourceString(), evt.Timestamp)
	} else if evt.Type == events.ReceiptTypeDelivered {
		log.Infof("%s was delivered to %s at %s", evt.MessageIDs[0], evt.SourceString(), evt.Timestamp)
	}
}

func handlePresence(evt *events.Presence) {
	if evt.Unavailable {
		if evt.LastSeen.IsZero() {
			log.Infof("%s is now offline", evt.From)
		} else {
			log.Infof("%s is now offline (last seen: %s)", evt.From, evt.LastSeen)
		}
	} else {
		log.Infof("%s is now online", evt.From)
	}
}

func handleHistorySync(evt *events.HistorySync) {
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
}

func handleAppState(evt *events.AppState) {
	log.Debugf("App state event: %+v / %+v", evt.Index, evt.SyncActionValue)
}

func handleKeepAliveTimeout(evt *events.KeepAliveTimeout) {
	log.Debugf("Keepalive timeout event: %+v", evt)
}

func handleKeepAliveRestored(evt *events.KeepAliveRestored) {
	log.Debugf("Keepalive restored")
}

func getFileExtension(mimeType string) string {
	exts, _ := mime.ExtensionsByType(mimeType)
	return exts[0]
}
