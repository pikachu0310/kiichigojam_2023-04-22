package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	Type string `json:"type"`
	ID   int    `json:"id"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Name string `json:"name"`
}

type JsonData struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

var (
	players      = make(map[int]*Player)
	playersMutex = sync.Mutex{}
	playerID     = 0
	upgrader     = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func main() {
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/", returnIndex)
	http.HandleFunc("/wasm_exec.js", returnWasmExec)
	http.HandleFunc("/client.wasm", returnClient)

	go handlePlayerUpdates()

	fmt.Println("Server listening on :8081")
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading connection:", err)
		return
	}
	defer conn.Close()

	go func() {
		for {
			playersMutex.Lock()
			copyPlayers := make(map[int]Player)
			for k, v := range players {
				copyPlayers[k] = *v
			}
			playersMutex.Unlock()

			jsonData := JsonData{
				Type: "players",
				Data: copyPlayers,
			}

			err := conn.WriteJSON(jsonData)
			if err != nil {
				fmt.Println("Error writing JSON:", err)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	for {
		var p Player
		err := conn.ReadJSON(&p)
		if err != nil {
			fmt.Println("Error reading JSON:", err)
			break
		}

		playersMutex.Lock()
		players[p.ID] = &p
		playersMutex.Unlock()
	}
}

func returnIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Serving index.html")
	http.ServeFile(w, r, "index.html")
}

func returnWasmExec(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Serving wasm_exec.js")
	http.ServeFile(w, r, "wasm_exec.js")
}

func returnClient(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Serving client.wasm")
	http.ServeFile(w, r, "client.wasm")
}

func handlePlayerUpdates() {
	for {
		playersMutex.Lock()
		for id, p := range players {
			fmt.Printf("Player ID: %d, X: %d, Y: %d\n", id, p.X, p.Y)
		}
		playersMutex.Unlock()
		time.Sleep(1000 * time.Millisecond)
	}
}
