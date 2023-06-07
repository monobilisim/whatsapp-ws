package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	var err error
	wsConn, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Failed to upgrade connection: %v", err)
		return
	}
	defer wsConn.Close()

	for {
		messageType, message, err := wsConn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				log.Errorf("Failed to read message: %v", err)
			}
			break
		}

		if messageType == websocket.TextMessage {
			cmd := strings.TrimSpace(string(message))
			if len(cmd) > 0 {
				go handleWsCmd(cmd)
			}
		}
	}
}

// ServeStatus returns the current status of the client
func serveStatus(w http.ResponseWriter, r *http.Request) {
	if cli.IsLoggedIn() {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cli.Store.ID.String()))
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

// Parse command, return command and arguments
func parseCmd(cmd string) (string, []string, error) {
	tokens := strings.Fields(cmd)

	if len(tokens) < 2 {
		return "", nil, errors.New("invalid command format")
	}

	command := strings.ToLower(tokens[0])
	args := tokens[1:]

	return command, args, nil
}

func handleWsCmd(cmd string) {
	command, args, err := parseCmd(cmd)

	if err != nil {
		log.Errorf(err.Error())
		return
	}

	go handleCmd(command, args)
}
