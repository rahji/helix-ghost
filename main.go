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
	Port int `kong:"default=4001,name='http-port',help='HTTP port'"`
	// Helix   string `kong:"default='hx',name='helix-command',help='Helix command'"`
	Verbose bool `kong:"name='verbose',help='Show extra output in the terminal'"`
}

type GhostText struct {
	Filename string
	Title    string `json:"title"`
	Text     string `json:"text"`
}

func main() {
	var cli CLIFlags
	kong.Parse(&cli)

	// Handle the initial GET request on port specified by Port flag
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		// Find an available port for WebSocket
		wsListener, err := net.Listen("tcp", "127.0.0.1:0") // Random available port
		if err != nil {
			http.Error(w, "Failed to find available port", http.StatusInternalServerError)
			return
		}

		wsPort := wsListener.Addr().(*net.TCPAddr).Port
		log.Printf("WebSocket will listen on port %d", wsPort)

		// Start the WebSocket listener in the background
		go handleWebSocketConnections(wsListener)

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

func handleWebSocketConnections(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Failed to accept connection:", err)
			continue
		}

		// Perform WebSocket upgrade
		_, err = ws.Upgrade(conn)
		if err != nil {
			log.Println("WebSocket upgrade failed:", err)
			conn.Close()
			continue
		}

		log.Println("WebSocket connection established")

		tmpFile, _ := os.CreateTemp(os.TempDir(), "*.txt")
		tmpFilename := tmpFile.Name()

		go func(c net.Conn) {

			defer c.Close()
			for {
				// msg, op, err := wsutil.ReadClientData(c)
				msg, _, err := wsutil.ReadClientData(c)
				if err != nil {
					log.Println("Read error:", err)
					return
				}

				var incoming GhostText
				err = json.Unmarshal(msg, &incoming)
				if err != nil {
					log.Println("Error unmarshalling payload:", err)
					return
				}

				// log.Printf("Title: %s", incoming.Title)
				log.Printf("message: %s", string(msg))
				_ = os.WriteFile(tmpFilename, []byte(incoming.Text), 0o644)

				// xxx check to see if file has changed and send new text to ws client

				// err = wsutil.WriteServerMessage(c, op, []byte("Echo: "+string(msg)))
				// if err != nil {
				// 	log.Println("Write error:", err)
				// 	return
				// }
			}
		}(conn)

		log.Println("Starting helix with temp file: " + tmpFilename)
		err = openEditor("hx", tmpFilename)
		if err != nil {
			panic(err)
		}
		// cleanup after editor closes
		os.Remove(tmpFilename)
		conn.Close()
	}
}

// openEditor opens an editor. The arguments are the command name and the filename to open
func openEditor(editor string, fn string) error {
	cmd := exec.Command(editor, fn)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
