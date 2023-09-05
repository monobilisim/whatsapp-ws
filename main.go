// Copyright (c) 2021 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var (
	cli              *whatsmeow.Client
	log              waLog.Logger
	logLevel         = "INFO"
	debugLogs        = flag.Bool("debug", false, "Enable debug logs?")
	dbDialect        = flag.String("db-dialect", "sqlite3", "Database dialect (sqlite3 or postgres)")
	dbAddress        = flag.String("db-address", "file:mdtest.db?sslmode=disable", "Database address")
	requestFullSync  = flag.Bool("request-full-sync", false, "Request full (1 year) history sync when logging in?")
	wsPort           = flag.String("ws-port", "8080", "WebSocket port")
	chatLogDBAddress = flag.String("chatlog-db-address", "postgresql://local@localhost/testing?sslmode=disable", "Chat log database address")
	dirPtr           = flag.String("data-dir", "/opt/whatsapp/data", "Directory to serve files from")
	minioEndpoint    = flag.String("minio-endpoint", "localhost:9000", "Minio endpoint")
	useSSLMinio      = flag.Bool("use-ssl-minio", true, "Use SSL for Minio?")
	minioBucket      = flag.String("minio-bucket", "whatsmeow", "Minio bucket")
	pairRejectChan   = make(chan bool, 1)
	wsConn           *websocket.Conn
	storeContainer   *sqlstore.Container
	minioClient      *minio.Client
	db               *sql.DB
	qrStr            string
)

func main() {
	waBinary.IndentXML = true
	flag.Parse()

	if *debugLogs {
		logLevel = "DEBUG"
	}
	if *requestFullSync {
		store.DeviceProps.RequireFullSync = proto.Bool(true)
	}
	log = waLog.Stdout("Main", logLevel, true)

	var err error
	dbLog := waLog.Stdout("Database", logLevel, true)
	storeContainer, err = sqlstore.New(*dbDialect, *dbAddress, dbLog)
	if err != nil {
		log.Errorf("Failed to connect to session database: %v", err)
		return
	}
	device, err := storeContainer.GetFirstDevice()
	if err != nil {
		log.Errorf("Failed to get device: %v", err)
		return
	}
	err = godotenv.Load("../credentials.env")

	if err != nil {
		log.Errorf("Error loading .env file")
	}

	accessKeyID := os.Getenv("ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("SECRET_ACCESS_KEY")
	*minioEndpoint = os.Getenv("MINIO_ENDPOINT")

	log.Infof("Connecting to Minio")
	log.Infof("Endpoint: %s", *minioEndpoint)
	minioClient, err = minio.New(*minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: *useSSLMinio,
	})

	if err != nil {
		log.Errorf("Failed to connect to Minio: %v", err)
		return
	}
	// Serve WebSocket endpoint
	http.HandleFunc("/ws", serveWs)
	http.HandleFunc("/status", serveStatus)
	http.HandleFunc("/qr", serveQR)
	http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		uploadHandler(w, r, *dirPtr)
	})

	go func() {
		log.Infof("Starting WebSocket server")
		err := http.ListenAndServe(":"+*wsPort, nil)
		if err != nil {
			log.Errorf("Failed to start WebSocket server: %v", err)
			os.Exit(1)
		}
	}()

	cli = whatsmeow.NewClient(device, waLog.Stdout("Client", logLevel, true))
	log.Infof("Device: %v", cli.Store.ID)
	var isWaitingForPair atomic.Bool
	cli.PrePairCallback = func(jid types.JID, platform, businessName string) bool {
		isWaitingForPair.Store(true)
		defer isWaitingForPair.Store(false)
		log.Infof("Pairing %s (platform: %q, business name: %q). Type r within 3 seconds to reject pair", jid, platform, businessName)
		select {
		case reject := <-pairRejectChan:
			if reject {
				log.Infof("Rejecting pair")
				return false
			}
		case <-time.After(3 * time.Second):
		}
		log.Infof("Accepting pair")
		return true
	}

	// Connect to chatlog database
	db, err = sql.Open("postgres", *chatLogDBAddress)
	if err != nil {
		log.Errorf("Failed to connect chatlog database: %v", err)
		return
	}
	defer db.Close()
	ch, err := cli.GetQRChannel(context.Background())
	if err != nil {
		// This error means that we're already logged in, so ignore it.
		if !errors.Is(err, whatsmeow.ErrQRStoreContainsID) {
			log.Errorf("Failed to get QR channel: %v", err)
		}
	} else {
		go func() {
			for evt := range ch {
				if evt.Event == "code" {
					qrStr = evt.Code
					qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)

				} else {
					log.Infof("QR channel result: %s", evt.Event)
				}
			}
		}()
	}

	cli.AddEventHandler(eventHandler)
	err = cli.Connect()
	if err != nil {
		log.Errorf("Failed to connect: %v", err)
		return
	}

	c := make(chan os.Signal, 1)
	input := make(chan string)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		defer close(input)
		scan := bufio.NewScanner(os.Stdin)
		for scan.Scan() {
			line := strings.TrimSpace(scan.Text())
			if len(line) > 0 {
				input <- line
			}
		}
	}()
	cli.SetStatusMessage("MAS Hukuk Bürosu")
	for {
		select {
		case <-c:
			log.Infof("Interrupt received, exiting")
			cli.Disconnect()
			return
		case cmd := <-input:
			if len(cmd) == 0 {
				log.Infof("Stdin closed, exiting")
				cli.Disconnect()
				return
			}
			if isWaitingForPair.Load() {
				if cmd == "r" {
					pairRejectChan <- true
				} else if cmd == "a" {
					pairRejectChan <- false
				}
				continue
			}
			var command Command
			args := strings.Fields(cmd)
			if len(args) == 0 {
				continue
			}
			command.Cmd = strings.ToLower(args[0])
			command.Arguments = args[1:]

			go handleCmd(command)
		}
	}
}

var historySyncID int32
var startupTime = time.Now().Unix()
