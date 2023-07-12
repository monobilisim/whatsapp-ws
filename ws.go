package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/mdp/qrterminal/v3"
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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	if cli.IsLoggedIn() {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cli.Store.ID.String()))
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)
}

func serveQR(w http.ResponseWriter, r *http.Request) {
	if cli.IsLoggedIn() {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	qrterminal.GenerateHalfBlock(qrStr, qrterminal.L, w)
}

func uploadHandler(w http.ResponseWriter, r *http.Request, uploadDir string) {
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	JID := r.FormValue("jid")
	userID, err := strconv.Atoi(r.FormValue("user_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mimeType := http.DetectContentType(data)

	// if mimeType is image, use handleSendImage, if its document, use handleSendDocument
	if mimeType[:5] == "image" {
		handleSendImage(JID, handler.Filename, userID, data)
	} else {
		handleSendDocument(JID, handler.Filename, userID, data)
	}

	log.Infof("Uploaded file %s to %s, mimetype: %s", handler.Filename, JID, mimeType)
	fmt.Fprintf(w, "File uploaded successfully\n")
}
