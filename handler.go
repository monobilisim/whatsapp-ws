package main

import (
	"strings"

	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
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
		handleSendTextMessage(command.Arguments, command.UserID)
	case "markread":
		handleMarkRead(command.Arguments)
	case "sendimage":
		handleSendImage(command.Arguments, command.UserID)
	case "senddocument":
		handleSendDocument(command.Arguments, command.UserID)
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
