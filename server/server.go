package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn       *websocket.Conn
	Send       chan []byte
	GameID     string
	Player     int
	PlayAgain  bool
	Disconnect chan struct{}
}

var (
	clients   = make(map[*Client]bool)
	games     = make(map[string]*Game)
	gameLocks = make(map[string]*sync.Mutex)
	upgrader  = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Connect 4 Server is running. Use the client application to connect.")
	})

	http.HandleFunc("/ws", handleConnections)

	fmt.Println("Server started on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to websocket:", err)
		return
	}

	client := &Client{
		Conn:       ws,
		Send:       make(chan []byte),
		Disconnect: make(chan struct{}),
	}

	gameID := r.URL.Query().Get("game_id")
	if gameID == "" {
		gameID = "game-1"
	}
	client.GameID = gameID

	gameLock := getGameLock(gameID)
	gameLock.Lock()

	existingPlayers := getPlayersInGame(gameID)
	if len(existingPlayers) == 0 {
		client.Player = 1
	} else if len(existingPlayers) == 1 {
		client.Player = otherPlayer(existingPlayers[0])
	} else {
		gameLock.Unlock()
		ws.WriteMessage(websocket.TextMessage, []byte("Game is full"))
		ws.Close()
		return
	}

	clients[client] = true

	if _, ok := games[gameID]; !ok {
		games[gameID] = NewGame()
	}

	gameLock.Unlock()

	go client.writePump()

	initialMsg := map[string]interface{}{
		"type":   "init",
		"player": client.Player,
	}
	msgBytes, _ := json.Marshal(initialMsg)
	client.Send <- msgBytes

	sendGameState(client)

	defer func() {
		ws.Close()
		handleClientDisconnect(client)
	}()

	for {
		var msg map[string]interface{}
		err := ws.ReadJSON(&msg)
		if err != nil {
			fmt.Println("Error reading JSON:", err)
			break
		}

		handleMessage(client, msg)
	}
}

func handleMessage(client *Client, msg map[string]interface{}) {
	gameID := client.GameID
	gameLock := getGameLock(gameID)
	gameLock.Lock()
	defer gameLock.Unlock()

	game := games[gameID]

	switch msg["type"] {
	case "move":
		column := int(msg["column"].(float64))

		if game.CurrentTurn != client.Player || game.IsOver {
			return
		}

		success := game.DropPiece(column)
		if success {
			broadcastGameState(gameID)
		}
	case "play_again":
		client.PlayAgain = true
		if allPlayersReady(gameID) {
			resetGame(gameID)
		}
	}
}

func broadcastGameState(gameID string) {
	game := games[gameID]
	gameState, _ := game.ToJSON()

	msg := map[string]interface{}{
		"type": "game_state",
		"game": json.RawMessage(gameState),
	}
	msgBytes, _ := json.Marshal(msg)

	for client := range clients {
		if client.GameID == gameID {
			client.Send <- msgBytes
		}
	}
}

func sendGameState(client *Client) {
	game := games[client.GameID]
	gameState, _ := game.ToJSON()

	msg := map[string]interface{}{
		"type": "game_state",
		"game": json.RawMessage(gameState),
	}
	msgBytes, _ := json.Marshal(msg)
	client.Send <- msgBytes
}

func (c *Client) writePump() {
	defer c.Conn.Close()
	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.WriteMessage(websocket.TextMessage, msg)
		case <-c.Disconnect:
			return
		}
	}
}

func handleClientDisconnect(client *Client) {
	fmt.Printf("Client %d disconnected\n", client.Player)
	delete(clients, client)
	close(client.Disconnect)

	gameID := client.GameID
	gameLock := getGameLock(gameID)
	gameLock.Lock()
	defer gameLock.Unlock()

	game := games[gameID]

	remainingClients := getClientsInGame(gameID)

	if len(remainingClients) == 0 {
		delete(games, gameID)
		delete(gameLocks, gameID)
	} else if len(remainingClients) == 1 {
		remainingClient := remainingClients[0]
		game.IsOver = false
		game.Winner = 0
		game.CurrentTurn = remainingClient.Player
		game.Board = [6][7]int{}
		broadcastGameState(gameID)
	} else {
		if game.CurrentTurn == client.Player {
			game.CurrentTurn = otherPlayer(client.Player)
			broadcastGameState(gameID)
		}
	}
}

func otherPlayer(player int) int {
	if player == 1 {
		return 2
	}
	return 1
}

func getClientsInGame(gameID string) []*Client {
	var gameClients []*Client
	for client := range clients {
		if client.GameID == gameID {
			gameClients = append(gameClients, client)
		}
	}
	return gameClients
}

func getPlayersInGame(gameID string) []int {
	var players []int
	for client := range clients {
		if client.GameID == gameID {
			players = append(players, client.Player)
		}
	}
	return players
}

func allPlayersReady(gameID string) bool {
	for client := range clients {
		if client.GameID == gameID && !client.PlayAgain {
			return false
		}
	}
	return true
}

func resetGame(gameID string) {
	games[gameID] = NewGame()

	for client := range clients {
		if client.GameID == gameID {
			client.PlayAgain = false
		}
	}

	msg := map[string]interface{}{
		"type": "reset",
	}
	msgBytes, _ := json.Marshal(msg)
	for client := range clients {
		if client.GameID == gameID {
			client.Send <- msgBytes
			sendGameState(client)
		}
	}
}

func getGameLock(gameID string) *sync.Mutex {
	lock, exists := gameLocks[gameID]
	if !exists {
		lock = &sync.Mutex{}
		gameLocks[gameID] = lock
	}
	return lock
}
