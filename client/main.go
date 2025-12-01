package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func main() {
	a := app.New()
	w := a.NewWindow("Connect 4")
	w.Resize(fyne.NewSize(700, 600))

	showMainMenu(w)

	w.Show()
	a.Run()
}

func showMainMenu(w fyne.Window) {
	title := canvas.NewText("Connect 4", color.White)
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter
	title.TextSize = 32

	subtitle := canvas.NewText("Choose a mode", color.RGBA{200, 200, 200, 255})
	subtitle.Alignment = fyne.TextAlignCenter
	subtitle.TextSize = 18

	onlineBtn := widget.NewButton("Online Multiplayer", func() {
		startOnlineGame(w)
	})

	difficultyLabel := widget.NewLabel("AI Difficulty")
	difficultySelect := widget.NewSelect([]string{
		DifficultyEasy,
		DifficultyMedium,
		DifficultyHard,
	}, nil)
	difficultySelect.Selected = DifficultyMedium

	singlePlayerBtn := widget.NewButton("Single Player vs AI", func() {
		difficulty := difficultySelect.Selected
		if difficulty == "" {
			difficulty = DifficultyMedium
		}
		startAIGame(w, difficulty)
	})

	buttons := container.NewVBox(
		layout.NewSpacer(),
		title,
		subtitle,
		widget.NewSeparator(),
		onlineBtn,
		layout.NewSpacer(),
		difficultyLabel,
		difficultySelect,
		singlePlayerBtn,
		layout.NewSpacer(),
	)

	windowBackground := canvas.NewRectangle(color.RGBA{15, 27, 39, 255})

	content := container.NewMax(
		windowBackground,
		container.NewPadded(buttons),
	)

	w.SetContent(content)
}

// startOnlineGame keeps the original multiplayer behavior, connecting
// to the remote websocket server and syncing the board.
func startOnlineGame(w fyne.Window) {
	game := &Game{}
	playerNumber := 0
	playAgainConfirmed := false

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, "ws://4.255.33.74/ws", nil)
	if err != nil {
		log.Println("Error connecting to WebSocket:", err)
		dialog.ShowError(err, w)
		return
	}

	statusLabel, cells, content := buildGameUI()

	for i := 0; i < 6; i++ {
		for j := 0; j < 7; j++ {
			col := j
			cells[i][j].SetOnTapped(func(_ int) {
				handleBoardClickOnline(game, col, conn, playerNumber)
			})
		}
	}

	statusLabel.SetText("Connecting to game...")

	w.SetContent(content)

	w.SetOnClosed(func() {
		_ = conn.Close(websocket.StatusNormalClosure, "window closed")
	})

	go func() {
		for {
			var msg map[string]interface{}
			err := wsjson.Read(ctx, conn, &msg)
			if err != nil {
				log.Println("Error reading message:", err)
				return
			}

			switch msg["type"] {
			case "init":
				playerNumber = int(msg["player"].(float64))
				statusLabel.SetText(fmt.Sprintf("You are Player %d", playerNumber))
			case "game_state":
				gameData, _ := json.Marshal(msg["game"])
				game.FromJSON(gameData)
				updateBoardUI(game, cells)
				updateStatusOnline(game, playerNumber, statusLabel)

				if game.IsOver && !playAgainConfirmed {
					promptPlayAgainOnline(w, conn, &playAgainConfirmed)
				}
			case "reset":
				game = &Game{}
				playAgainConfirmed = false
				updateBoardUI(game, cells)
				updateStatusOnline(game, playerNumber, statusLabel)
			}
		}
	}()
}

// startAIGame runs a full single-player game against an AI opponent
// with the selected difficulty, using the local Game logic.
func startAIGame(w fyne.Window, difficulty string) {
	game := NewGame()
	humanPlayer := 1
	aiPlayer := 2

	statusLabel, cells, content := buildGameUI()

	for i := 0; i < 6; i++ {
		for j := 0; j < 7; j++ {
			col := j
			cells[i][j].SetOnTapped(func(_ int) {
				handleBoardClickLocal(game, col, cells, statusLabel, difficulty, humanPlayer, aiPlayer, w)
			})
		}
	}

	statusLabel.SetText(fmt.Sprintf("Single Player - %s AI. Your turn.", difficulty))
	w.SetContent(content)
}

func buildGameUI() (*widget.Label, [][]*Slot, fyne.CanvasObject) {
	statusLabel := widget.NewLabel("")
	statusLabel.Alignment = fyne.TextAlignCenter
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	var cells [][]*Slot
	grid := container.NewGridWithColumns(7)
	cells = make([][]*Slot, 6)
	for i := 0; i < 6; i++ {
		cells[i] = make([]*Slot, 7)
		for j := 0; j < 7; j++ {
			row, col := i, j

			slot := NewSlot(row, col, func(int) {})
			cells[i][j] = slot
			grid.Add(slot)
		}
	}

	paddedGrid := container.NewVBox(
		layout.NewSpacer(),
		grid,
		layout.NewSpacer(),
	)

	boardBackground := canvas.NewRectangle(color.RGBA{39, 56, 74, 255})

	board := container.NewMax(
		boardBackground,
		paddedGrid,
	)

	windowBackground := canvas.NewRectangle(color.RGBA{15, 27, 39, 255})

	content := container.NewMax(
		windowBackground,
		container.NewVBox(
			statusLabel,
			board,
		),
	)

	return statusLabel, cells, content
}

func handleBoardClickOnline(game *Game, column int, conn *websocket.Conn, playerNumber int) {
	if game.IsOver || game.CurrentTurn != playerNumber {
		return
	}

	move := map[string]interface{}{
		"type":   "move",
		"column": column,
	}
	err := wsjson.Write(context.Background(), conn, move)
	if err != nil {
		log.Println("Error sending message:", err)
		return
	}
}

func handleBoardClickLocal(game *Game, column int, cells [][]*Slot, statusLabel *widget.Label, difficulty string, humanPlayer, aiPlayer int, w fyne.Window) {
	if game.IsOver || game.CurrentTurn != humanPlayer {
		return
	}

	if !game.DropPiece(column) {
		return
	}

	updateBoardUI(game, cells)
	updateStatusLocal(game, humanPlayer, aiPlayer, statusLabel, difficulty)

	if game.IsOver {
		promptPlayAgainLocal(w, game, cells, statusLabel, difficulty, humanPlayer, aiPlayer)
		return
	}

	if game.CurrentTurn != aiPlayer || game.IsOver {
		return
	}

	aiMove := GetAIMove(game, difficulty, aiPlayer)
	if aiMove == -1 {
		game.IsOver = true
		updateStatusLocal(game, humanPlayer, aiPlayer, statusLabel, difficulty)
		promptPlayAgainLocal(w, game, cells, statusLabel, difficulty, humanPlayer, aiPlayer)
		return
	}

	game.DropPiece(aiMove)
	updateBoardUI(game, cells)
	updateStatusLocal(game, humanPlayer, aiPlayer, statusLabel, difficulty)

	if game.IsOver {
		promptPlayAgainLocal(w, game, cells, statusLabel, difficulty, humanPlayer, aiPlayer)
	}
}

func updateBoardUI(game *Game, cells [][]*Slot) {
	for i := 0; i < 6; i++ {
		for j := 0; j < 7; j++ {
			piece := int(game.Board[i][j])
			cell := cells[i][j]
			cell.SetPiece(piece)
		}
	}
}

func updateStatusOnline(game *Game, playerNumber int, statusLabel *widget.Label) {
	if game.IsOver {
		if game.Winner == 0 {
			statusLabel.SetText("It's a tie!")
		} else if game.Winner == playerNumber {
			statusLabel.SetText("You win!")
		} else {
			statusLabel.SetText("You lose!")
		}
	} else {
		if game.CurrentTurn == playerNumber {
			statusLabel.SetText("Your turn")
		} else {
			statusLabel.SetText("Opponent's turn")
		}
	}
}

func updateStatusLocal(game *Game, humanPlayer, aiPlayer int, statusLabel *widget.Label, difficulty string) {
	if game.IsOver {
		if game.Winner == 0 {
			statusLabel.SetText(fmt.Sprintf("Single Player - %s AI. It's a tie!", difficulty))
		} else if game.Winner == humanPlayer {
			statusLabel.SetText(fmt.Sprintf("Single Player - %s AI. You win!", difficulty))
		} else if game.Winner == aiPlayer {
			statusLabel.SetText(fmt.Sprintf("Single Player - %s AI. You lose!", difficulty))
		}
	} else {
		if game.CurrentTurn == humanPlayer {
			statusLabel.SetText(fmt.Sprintf("Single Player - %s AI. Your turn.", difficulty))
		} else if game.CurrentTurn == aiPlayer {
			statusLabel.SetText(fmt.Sprintf("Single Player - %s AI. AI thinking...", difficulty))
		}
	}
}

func promptPlayAgainOnline(w fyne.Window, conn *websocket.Conn, playAgainConfirmed *bool) {
	dialog.ShowConfirm("Play Again", "Do you want to play again?", func(confirmed bool) {
		if confirmed {
			*playAgainConfirmed = true
			msg := map[string]interface{}{
				"type": "play_again",
			}
			err := wsjson.Write(context.Background(), conn, msg)
			if err != nil {
				log.Println("Error sending play again message:", err)
				return
			}
		} else {
			err := conn.Close(websocket.StatusNormalClosure, "User chose not to play again")
			if err != nil {
				log.Println("Error closing connection:", err)
			}
			w.Close()
		}
	}, w)
}

func promptPlayAgainLocal(w fyne.Window, game *Game, cells [][]*Slot, statusLabel *widget.Label, difficulty string, humanPlayer, aiPlayer int) {
	dialog.ShowConfirm("Play Again", "Do you want to play again?", func(confirmed bool) {
		if confirmed {
			newGame := NewGame()
			*game = *newGame
			updateBoardUI(game, cells)
			updateStatusLocal(game, humanPlayer, aiPlayer, statusLabel, difficulty)
		} else {
			w.Close()
		}
	}, w)
}

type Slot struct {
	widget.BaseWidget
	circle   *canvas.Circle
	piece    int
	row      int
	col      int
	onTapped func(col int)
}

func NewSlot(row, col int, onTapped func(col int)) *Slot {
	s := &Slot{
		circle:   canvas.NewCircle(color.RGBA{15, 27, 39, 255}),
		piece:    0,
		row:      row,
		col:      col,
		onTapped: onTapped,
	}
	s.ExtendBaseWidget(s)
	return s
}

func (s *Slot) CreateRenderer() fyne.WidgetRenderer {
	return &slotRenderer{
		slot:    s,
		objects: []fyne.CanvasObject{s.circle},
	}
}

func (s *Slot) Tapped(*fyne.PointEvent) {
	s.onTapped(s.col)
}

func (s *Slot) TappedSecondary(*fyne.PointEvent) {}

func (s *Slot) SetOnTapped(handler func(col int)) {
	s.onTapped = handler
}

type slotRenderer struct {
	slot    *Slot
	objects []fyne.CanvasObject
}

func (r *slotRenderer) Layout(size fyne.Size) {
	padding := float32(5)
	innerSize := fyne.NewSize(size.Width-padding*2, size.Height-padding*2)
	r.slot.circle.Resize(innerSize)
	r.slot.circle.Move(fyne.NewPos(padding, padding))
}

func (r *slotRenderer) MinSize() fyne.Size {
	return fyne.NewSize(60, 60)
}

func (r *slotRenderer) Refresh() {
	r.slot.circle.Refresh()
}

func (r *slotRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

func (r *slotRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *slotRenderer) Destroy() {}

func (s *Slot) SetPiece(piece int) {
	s.piece = piece
	switch piece {
	case 0:
		s.circle.FillColor = color.RGBA{15, 27, 39, 255}
	case 1:
		s.circle.FillColor = color.RGBA{26, 188, 157, 255}
	case 2:
		s.circle.FillColor = color.RGBA{239, 102, 119, 255}
	}
	s.circle.Refresh()
}
