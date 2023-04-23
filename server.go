package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Player struct {
	ID int
	X  int
	Y  int
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
	err := http.ListenAndServe(":8080", nil)
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
			for id, p := range players {
				err := conn.WriteJSON(p)
				if err != nil {
					fmt.Println("Error writing JSON:", err)
					return
				}
				fmt.Printf("Sent player %d\n", id)
				time.Sleep(20 * time.Millisecond)
			}
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