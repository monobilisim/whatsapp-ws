package main

import (
	"io"
	"net/http"
	"strconv"
	"strings"

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
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		handleError(w, http.StatusBadRequest, "Failed to parse multipart form", err)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		handleError(w, http.StatusBadRequest, "Failed to retrieve file from request", err)
		return
	}
	defer file.Close()

	JID := r.FormValue("jid")
	userID, err := strconv.Atoi(r.FormValue("user_id"))
	if err != nil {
		handleError(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		handleError(w, http.StatusInternalServerError, "Failed to read file data", err)
		return
	}

	mimeType := http.DetectContentType(data)

	var responseID string
	if strings.HasPrefix(mimeType, "image/") {
		responseID, err = handleSendImage(JID, handler.Filename, userID, data)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to handle image upload", err)
			return
		}
	} else {
		responseID, err = handleSendDocument(JID, handler.Filename, userID, data)
		if err != nil {
			handleError(w, http.StatusInternalServerError, "Failed to handle document upload", err)
			return
		}
	}

	log.Infof("Uploaded file %s to %s, mimetype: %s", handler.Filename, JID, mimeType)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(responseID))
}

func handleError(w http.ResponseWriter, statusCode int, message string, err error) {
	log.Errorf("%s: %v", message, err)
	http.Error(w, message, statusCode)
}
