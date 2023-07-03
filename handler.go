package main

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"strings"
	"sync/atomic"

	"go.mau.fi/whatsmeow/appstate"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

type Message struct {
	Jid  string
	Body string
	Sent bool
}

func handleCmd(cmd Command) {
	switch cmd.Cmd {
	case "isloggedin":
		handleIsLoggedIn()
	case "checkuser":
		handleCheckUser(cmd.Args)
	case "send":
		handleSendMessage(cmd.Args)
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

	if img := evt.Message.GetImageMessage(); img != nil {
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

	if err := InsertMessageHistory(evt.Info.ID, cli.Store.ID.String(), remotejid, evt.Info.Type, msgContent, evt.Info.Timestamp, evt.Info.MessageSource.IsFromMe); err != nil {
		log.Errorf("Failed to insert message history: %v", err)
	}

	if err := InsertOrUpdateLastMessage(evt.Info.ID, cli.Store.ID.String(), remotejid, evt.Info.Type, msgContent, evt.Info.Timestamp, evt.Info.MessageSource.IsFromMe); err != nil {
		log.Errorf("Failed to insert last messages: %v", err)
	}

	m := Message{remotejid, msgContent, evt.Info.MessageSource.IsFromMe}
	wsConn.WriteJSON(m)
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
}

func handleKeepAliveRestored(evt *events.KeepAliveRestored) {
	log.Debugf("Keepalive restored")
}

func handleIsLoggedIn() {
	log.Infof("Checking if logged in...")
	log.Infof("Logged in: %t", cli.IsLoggedIn())
}

func handleCheckUser(params []string) {
	if len(params) < 1 {
		log.Errorf("Usage: checkuser <phone numbers...>")
		return
	}

	resp, err := cli.IsOnWhatsApp(params)
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

	if err := InsertMessageHistory(resp.ID, cli.Store.ID.String(), recipient.String(), "text", msg.GetConversation(), resp.Timestamp, true); err != nil {
		log.Errorf("Error inserting message history: %v", err)
	}

	if err := InsertOrUpdateLastMessage(resp.ID, cli.Store.ID.String(), recipient.String(), "text", msg.GetConversation(), resp.Timestamp, true); err != nil {
		log.Errorf("Error inserting last messages: %v", err)
	}

	m := Message{recipient.String(), msg.GetConversation(), true}
	wsConn.WriteJSON(m)
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
