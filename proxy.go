package main

import (
	"fmt"
	"log"
	"net/http"

	"flag"
	"time"

	"go/build"
	"os"

	"code.google.com/p/go.net/websocket"
)

const basePkg = "github.com/chlu/webcamproxy"

// --------------------------

type Message []byte

type MessageSource interface {
	Initialize() error
	ReadMessage() (Message, error)
}

// --------------------------

const MaxInstacamFrameLength = 20000
const MessageBufferSize = 1024

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

type RandomMessageSource struct {
}

func (s *RandomMessageSource) Initialize() error {
	return nil
}

func (s *RandomMessageSource) ReadMessage() (Message, error) {
	msg := make([]byte, MaxInstacamFrameLength)
	// TODO add some randomness

	time.Sleep(100 * time.Millisecond)
	return msg, nil
}

// --------------------------

var webcamAddress = flag.String("webcam", "", "IP address of the webcam")
var address = flag.String("adress", ":8080", "Address for the proxy to listen")

var headerMessage *Message
var clients map[int]chan Message
var idSequence int = 0

var src MessageSource

// Write received messages from a channel to the websocket client
func FrameServer(conn *websocket.Conn) {
	// Set the PayloadType to binary
	conn.PayloadType = websocket.BinaryFrame

	// TODO Send first message as header to every new client
	// struct { char magic[4] = "jsmp"; unsigned short width, height; };

	queue := make(chan Message, MessageBufferSize)

	// Assign a unique id to the client
	idSequence++
	id := idSequence

	// TODO Guard this!
	clients[id] = queue

	defer func() {
		delete(clients, id)
	}()

	if _, err := conn.Write(*headerMessage); err != nil {
		return
	}

	// Read messages from the queue and write to the client
	for msg := range queue {
		if _, err := conn.Write(msg); err != nil {
			return
		}
	}
}

func readMessages() {
	for {
		if msg, err := src.ReadMessage(); err != nil {
			log.Printf("Error reading message from source: %v", err)
		} else {
			if headerMessage == nil {
				headerMessage = &msg
			}

			for _, queue := range clients {
				select {
				case queue <- msg:
					// No op
				default:
					log.Printf("Not send to client, blocked or closed?")
				}
			}
		}
	}
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

	src = &InstacamMessageSource{Url: "ws://" + *webcamAddress + ":80/ws"}

	err := src.Initialize()
	if err != nil {
		log.Fatal(err)
	}

	clients = make(map[int]chan Message)

	go readMessages()

	http.Handle("/ws", websocket.Handler(FrameServer))
	http.Handle("/", http.FileServer(http.Dir(getResourceRoot()+"/resources/")))
	http.ListenAndServe(*address, nil)
}
