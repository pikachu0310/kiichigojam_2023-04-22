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

	go handlePlayerUpdates()

	fmt.Println("Server listening on :8080")
	err := http.ListenAndServeTLS(":8080", "cert.pem", "key.pem", nil)
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
