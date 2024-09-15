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
	game := &Game{}
	playerNumber := 0
	playAgainConfirmed := false

	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, "ws://4.255.33.74/ws", nil)
	if err != nil {
		log.Println("Error connecting to WebSocket:", err)
		return
	}
	defer conn.Close(websocket.StatusInternalError, "Internal Error")

	a := app.New()
	w := a.NewWindow("Connect 4")
	w.Resize(fyne.NewSize(700, 600))

	statusLabel := widget.NewLabel("Connecting to game...")
	statusLabel.Alignment = fyne.TextAlignCenter
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	var cells [][]*Slot
	grid := container.NewGridWithColumns(7)
	cells = make([][]*Slot, 6)
	for i := 0; i < 6; i++ {
		cells[i] = make([]*Slot, 7)
		for j := 0; j < 7; j++ {
			row, col := i, j

			slot := NewSlot(row, col, func(col int) {
				handleBoardClick(game, col, conn, playerNumber)
			})
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

	w.SetContent(content)
	w.Show()

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
				updateStatus(game, playerNumber, statusLabel)

				if game.IsOver && !playAgainConfirmed {
					promptPlayAgain(w, conn, &playAgainConfirmed)
				}
			case "reset":
				game = &Game{}
				playAgainConfirmed = false
				updateBoardUI(game, cells)
				updateStatus(game, playerNumber, statusLabel)
			}
		}
	}()

	a.Run()
}

func handleBoardClick(game *Game, column int, conn *websocket.Conn, playerNumber int) {
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

func updateBoardUI(game *Game, cells [][]*Slot) {
	for i := 0; i < 6; i++ {
		for j := 0; j < 7; j++ {
			piece := int(game.Board[i][j])
			cell := cells[i][j]
			cell.SetPiece(piece)
		}
	}
}

func updateStatus(game *Game, playerNumber int, statusLabel *widget.Label) {
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

func promptPlayAgain(w fyne.Window, conn *websocket.Conn, playAgainConfirmed *bool) {
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
