package main

import (
	"net/http"

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

type Command struct {
	Cmd       string   `json:"cmd"`
	Arguments []string `json:"args"`
	UserID    int      `json:"user_id"`
}

// Handle incoming WebSocket connections, read json messages and pass them to the handleCmd function
func serveWs(w http.ResponseWriter, r *http.Request) {
	var err error
	wsConn, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("Failed to upgrade connection: %v", err)
		return
	}
	defer wsConn.Close()

	for {
		var cmd Command
		err := wsConn.ReadJSON(&cmd)
		if err != nil {
			log.Errorf("Failed to read json: %v", err)
			return
		}
		handleCmd(cmd)
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
	w.WriteHeader(http.StatusServiceUnavailable)
}
