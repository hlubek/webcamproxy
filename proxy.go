package main

import (
	"fmt"
	"log"
	"net/http"

	"flag"

	"go/build"
	"os"

	"code.google.com/p/go.net/websocket"
)

const basePkg = "github.com/chlu/webcamproxy"

// --------------------------

const MessageBufferSize = 1024

type Message []byte

type MessageSource interface {
	Initialize() error
	ReadMessage() (Message, error)
}

type Client struct {
	queue chan *Message
}

func NewClient() *Client {
	queue := make(chan *Message, MessageBufferSize)
	return &Client{queue}
}

// --------------------------

const MaxInstacamFrameLength = 20000

type InstacamMessageSource struct {
	// "ws://1.2.3.4:80/ws"
	Url     string
	camConn *websocket.Conn
}

func (s *InstacamMessageSource) Initialize() error {
	if ws, err := websocket.Dial(s.Url, "", "http://localhost/"); err != nil {
		return fmt.Errorf("Error initializing Instacam stream: %v", err)
	} else {
		s.camConn = ws
	}
	return nil
}

func (s *InstacamMessageSource) ReadMessage() (Message, error) {
	msg := make([]byte, MaxInstacamFrameLength)
	if n, err := s.camConn.Read(msg); err != nil {
		return nil, err
	} else {
		if n == MaxInstacamFrameLength {
			return nil, fmt.Errorf("Frame size exceeded buffer (%d bytes)", MaxInstacamFrameLength)
		} else {
			return Message(msg[:n]), nil
		}
	}
	return nil, nil
}

// --------------------------

var webcamAddress = flag.String("webcam", "", "IP address of the webcam")
var address = flag.String("address", ":8080", "Address for the proxy to listen")
var verbose = flag.Bool("verbose", false, "Output debug messages")

var src MessageSource

var headerMessage *Message

// Map of clients by id to a channel of messages
var clients map[*Client]chan *Message

// Quit channel for the readMessages goroutine
var quitReading chan bool

var registerClient chan *Client
var removeClient chan *Client

// --------------------------

// Write received messages from a channel to the websocket client
func FrameServer(conn *websocket.Conn) {
	// Set the PayloadType to binary
	conn.PayloadType = websocket.BinaryFrame

	client := NewClient()

	registerClient <- client
	defer func() {
		removeClient <- client
	}()

	// Write the header as the first message to the client (MPEG size etc.)
	if _, err := conn.Write(*headerMessage); err != nil {
		return
	}

	// Read messages from the queue and write to the client
	for msg := range client.queue {
		if _, err := conn.Write(*msg); err != nil {
			return
		}
	}
}

func handleClients() {
	for {
		select {
		case client := <-registerClient:
			clients[client] = client.queue

			// If we added the first client, start reading of messages
			if len(clients) == 1 {
				go readMessages()
			}

			if *verbose {
				log.Printf("Client connected, %d clients total", len(clients))
			}
		case client := <-removeClient:
			delete(clients, client)

			if len(clients) == 0 {
				quitReading <- true
			}

			if *verbose {
				log.Printf("Client disconnected, %d clients total", len(clients))
			}
		}
	}
}

func readMessages() {
	if *verbose {
		log.Printf("Start reading messages")
	}
	for {
		select {
		case <-quitReading:
			if *verbose {
				log.Printf("Stop reading messages")
			}
			return
		default:
			if msg, err := src.ReadMessage(); err != nil {
				// TODO Handle connection problems to webcam, eg. call Initialize again after some time
				if *verbose {
					log.Printf("Error reading message from source: %v", err)
				}
			} else {
				for _, queue := range clients {
					select {
					case queue <- &msg:
						// No op
					default:
						if *verbose {
							log.Printf("Message not sent to client, blocked or closed?")
						}
					}
				}
			}
		}
	}
}

func readHeaderMessage() error {
	if msg, err := src.ReadMessage(); err != nil {
		return err
	} else {
		headerMessage = &msg
	}
	return nil
}

func getResourceRoot() string {
	p, err := build.Default.Import(basePkg, "", build.FindOnly)
	if err != nil {
		log.Fatalf("Couldn't find resource files: %v", err)
	}
	return p.Dir
}

func main() {
	flag.Parse()

	if *webcamAddress == "" {
		flag.Usage()
		os.Exit(0)
	}

	websocketUrl := "ws://" + *webcamAddress + ":80/ws"
	src = &InstacamMessageSource{Url: websocketUrl}
	if err := src.Initialize(); err != nil {
		log.Fatalf("Error initializing source: %v", err)
	} else {
		if *verbose {
			log.Printf("Connected to source %s", websocketUrl)
		}
	}
	if err := readHeaderMessage(); err != nil {
		log.Fatalf("Error reading header message from source: %v", err)
	}

	clients = make(map[*Client]chan *Message)
	quitReading = make(chan bool)
	registerClient = make(chan *Client)
	removeClient = make(chan *Client)

	go handleClients()

	http.Handle("/ws", websocket.Handler(FrameServer))
	http.Handle("/", http.FileServer(http.Dir(getResourceRoot()+"/resources/")))
	http.ListenAndServe(*address, nil)
}
