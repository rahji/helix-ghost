package main

// todo
// ✅ create temp file and fill with text from json payload
// ✅ open editor
// 3. watch file for updates and send to client as json
// 4. if editor exits, end websockets connection and delete temp file
//    if client ends websockets connection, kill editor and delete temp file
//    if another websockets connection is started, kill editor, delete temp file, start at step 1 again

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/alecthomas/kong"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type CLIFlags struct {
	Port    int    `kong:"default=4001,name='http-port',help='HTTP port'"`
	Editor  string `kong:"default='hx',name='editor',help='Editor command'"`
	Verbose bool   `kong:"name='verbose',help='Show extra output in the terminal'"`
}

var cli CLIFlags

type GhostText struct {
	Filename string
	Title    string `json:"title"`
	Text     string `json:"text"`
}

var GTSession GhostText

type FileChangeEvent struct {
	Filename string
	Content  []byte
}

func main() {
	kong.Parse(&cli)

	limiter := &ConnectionLimiter{}

	// set up the handler for an HTTP request from the client
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		if limiter.IsActive() {
			http.Error(w, "WebSocket connection already active", http.StatusServiceUnavailable)
			log.Println("Ignoring new connection: a ws connection is already active")
			return
		}

		// Find an available port for WebSocket
		wsListener, err := net.Listen("tcp", "127.0.0.1:0") // Random available port
		if err != nil {
			http.Error(w, "Failed to find available port", http.StatusInternalServerError)
			return
		}

		wsPort := wsListener.Addr().(*net.TCPAddr).Port
		log.Printf("WebSocket will listen on port %d", wsPort)

		go handleWebSockets(wsListener, limiter)

		// Respond to client with JSON payload described in GhostText PROTOCOL
		resp := map[string]interface{}{
			"WebSocketPort":   wsPort,
			"ProtocolVersion": 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

	})

	port := fmt.Sprintf(":%d", cli.Port)
	log.Println("HTTP server listening on port ", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

// handleWebSockets listens and intitiates websocket connections
func handleWebSockets(ln net.Listener, limiter *ConnectionLimiter) {

	limiter.SetActive(true)
	defer limiter.SetActive(false)

	conn, err := ln.Accept()
	if err != nil {
		log.Println("Failed to accept connection:", err)
		return
	}

	defer conn.Close()

	// Perform WebSocket upgrade
	_, err = ws.Upgrade(conn)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		conn.Close()
		return
	}

	log.Println("WebSocket connection established")

	// Read only the first message from the client, which contains the existing textarea contents.
	// We can't continuously read changes to the textarea since Helix has no way to update the buffer.
	// It also doesn't update the buffer when the file contents are changed from outside Helix!
	msg, _, err := wsutil.ReadClientData(conn)
	if err != nil {
		log.Println("Read error:", err)
		return
	}

	if err := json.Unmarshal(msg, &GTSession); err != nil {
		log.Println("Error unmarshalling payload:", err)
		return
	}

	tempFile, err := createTempFile(GTSession.Text)
	if err != nil {
		log.Fatal("Couldn't create temp file: ", err)
	}
	GTSession.Filename = tempFile.Name()

	// xxx file watcher goroutine needs to start to track changes and write them to the websocket!

	if err := openEditor(cli.Editor, GTSession.Filename); err != nil {
		log.Fatal("Couldn't open editor with temp file: ", err)
	}

}

// createTempFile creates a temp file and populates it with the provided string
func createTempFile(text string) (*os.File, error) {
	tempFile, err := os.CreateTemp(os.TempDir(), "*.txt")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(tempFile.Name(), []byte(text), 0o644); err != nil {
		return nil, err
	}
	return tempFile, nil
}

// openEditor opens an a specified file in an editor
func openEditor(editor string, fn string) error {
	cmd := exec.Command(editor, fn)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
