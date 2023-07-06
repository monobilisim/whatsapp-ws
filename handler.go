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

type Message struct {
	MessageID string
	Jid       string
	Body      string
	Sent      bool
}

func handleCmd(command Command) {
	switch command.Cmd {
	case "isloggedin":
		handleIsLoggedIn()
	case "checkuser":
		handleCheckUser(command.Arguments)
	case "send":
		handleSendMessage(command.Arguments, command.UserID)
	case "markread":
		handleMarkRead(command.Arguments)
	}
}

// Handler is a simple eventHandler for incoming events.
func eventHandler(rawEvt interface{}) {
	switch evt := rawEvt.(type) {
	case *events.AppStateSyncComplete:
		handleAppStateSyncComplete(evt)
	case *events.Connected, *events.PushNameSetting:
		handleConnectedOrPushNameSetting(evt)
	case *events.StreamReplaced:
		handleStreamReplaced(evt)
	case *events.Message:
		handleMessage(evt)
	case *events.Receipt:
		handleReceipt(evt)
	case *events.Presence:
		handlePresence(evt)
	case *events.HistorySync:
		handleHistorySync(evt)
	case *events.AppState:
		handleAppState(evt)
	case *events.KeepAliveTimeout:
		handleKeepAliveTimeout(evt)
	case *events.KeepAliveRestored:
		handleKeepAliveRestored(evt)
	}
}

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
		exts, _ := mime.ExtensionsByType(img.GetMimetype())
		extension = exts[0]
		path := fmt.Sprintf("%s%s", evt.Info.ID, extension)
		_ = os.WriteFile(path, data, 0600)

		path = fmt.Sprintf("%s%s", evt.Info.ID, ".jpg")
		err = os.WriteFile(path, img.GetJpegThumbnail(), 0600)

		if err != nil {
			log.Errorf("Failed to save image: %v", err)
			return
		}
		log.Infof("Saved image message to %s", path)
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
		err = os.WriteFile(path, data, 0600)
		if err != nil {
			log.Errorf("Failed to save document: %v", err)
			return
		}

		fileName = doc.GetFileName()

		path = fmt.Sprintf("%s%s", evt.Info.ID, ".jpg")
		err = os.WriteFile(path, doc.GetJpegThumbnail(), 0600)

		if err != nil {
			log.Errorf("Failed to save document: %v", err)
			return
		}
		log.Infof("Saved document message to %s", path)
	}

	if audio := evt.Message.GetAudioMessage(); audio != nil {
		data, err := cli.Download(audio)
		if err != nil {
			log.Errorf("Failed to download audio: %v", err)
			return
		}
		exts, _ := mime.ExtensionsByType(audio.GetMimetype())
		extension = exts[0]
		path := fmt.Sprintf("%s%s", evt.Info.ID, extension)
		err = os.WriteFile(path, data, 0600)
		if err != nil {
			log.Errorf("Failed to save audio: %v", err)
			return
		}

		log.Infof("Saved audio message to %s", path)
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

	remoteJid := evt.Info.MessageSource.Chat.String()

	if evt.Info.Category == "peer" {
		// Bunlar ilk login olunduğunda alınan sistem mesajları, veritabanına yazmayalım.
		// Örn: [Main INFO] Received message 469474B7AB6166188B238F2AD94F5A65 from 905015301816@s.whatsapp.net (pushname: MAS Hukuk, timestamp: 2023-06-21 12:16:33 +0300 +03, type: text, category: peer): protocolMessage:{type:INITIAL_SECURITY_NOTIFICATION_SETTING_SYNC initialSecurityNotificationSettingSync:{securityNotificationEnabled:false}}
		return
	}

	if err := insertMessages(evt.Info.ID, cli.Store.ID.String(), remoteJid, evt.Info.Type, msgContent, evt.Info.Timestamp, evt.Info.MessageSource.IsFromMe, extension, fileName, -1); err != nil {
		log.Errorf("Error inserting into messages: %v", err)
	}

	if err := insertLastMessages(evt.Info.ID, cli.Store.ID.String(), remoteJid, evt.Info.Type, msgContent, evt.Info.Timestamp, evt.Info.MessageSource.IsFromMe, extension, fileName, -1); err != nil {
		log.Errorf("Error inserting into last_messages: %v", err)
	}

	if wsConn != nil {
		m := Message{evt.Info.ID, remoteJid, msgContent, evt.Info.MessageSource.IsFromMe}
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

func handleSendMessage(args []string, user_id int) {
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

	if err := insertMessages(resp.ID, cli.Store.ID.String(), recipient.String(), "text", msg.GetConversation(), resp.Timestamp, true, "", "", user_id); err != nil {
		log.Errorf("Error inserting into messages: %v", err)
	}

	if err := insertLastMessages(resp.ID, cli.Store.ID.String(), recipient.String(), "text", msg.GetConversation(), resp.Timestamp, true, "", "", user_id); err != nil {
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

	message_id := args[0]
	remote_jid := args[1]

	sender, ok := parseJID(remote_jid)
	if !ok {
		return
	}

	timestamp := time.Now()

	err := cli.MarkRead([]string{message_id}, timestamp, sender, sender)
	if err != nil {
		log.Errorf("Error marking read: %v", err)
		return
	}
	log.Infof("MarkRead sent: %s %s %s", message_id, timestamp, sender)

	if err := markMessageRead(message_id, remote_jid, timestamp); err != nil {
		log.Errorf("Error marking message as read: %v", err)
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
