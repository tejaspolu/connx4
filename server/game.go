package main

import (
	"encoding/json"
)

type Game struct {
	Board       [6][7]int `json:"board"`
	CurrentTurn int       `json:"current_turn"`
	IsOver      bool      `json:"is_over"`
	Winner      int       `json:"winner"`
}

func NewGame() *Game {
	return &Game{
		Board:       [6][7]int{},
		CurrentTurn: 1,
		IsOver:      false,
		Winner:      0,
	}
}

func (g *Game) DropPiece(column int) bool {
	if column < 0 || column > 6 || g.IsOver {
		return false
	}

	for row := 5; row >= 0; row-- {
		if g.Board[row][column] == 0 {
			g.Board[row][column] = g.CurrentTurn
			g.checkWin(row, column)
			g.toggleTurn()
			return true
		}
	}
	return false
}

func (g *Game) toggleTurn() {
	if g.CurrentTurn == 1 {
		g.CurrentTurn = 2
	} else {
		g.CurrentTurn = 1
	}
}

func (g *Game) checkWin(row, col int) {
	directions := [][2]int{
		{0, 1},
		{1, 0},
		{1, 1},
		{1, -1},
	}

	for _, dir := range directions {
		count := 1
		count += g.countDirection(row, col, dir[0], dir[1])
		count += g.countDirection(row, col, -dir[0], -dir[1])

		if count >= 4 {
			g.IsOver = true
			g.Winner = g.CurrentTurn
			return
		}
	}

	if g.isBoardFull() {
		g.IsOver = true
	}
}

func (g *Game) countDirection(row, col, deltaRow, deltaCol int) int {
	count := 0
	player := g.CurrentTurn

	for {
		row += deltaRow
		col += deltaCol
		if row < 0 || row > 5 || col < 0 || col > 6 || g.Board[row][col] != player {
			break
		}
		count++
	}
	return count
}

func (g *Game) isBoardFull() bool {
	for col := 0; col < 7; col++ {
		if g.Board[0][col] == 0 {
			return false
		}
	}
	return true
}

func (g *Game) ToJSON() ([]byte, error) {
	return json.Marshal(g)
}

func (g *Game) FromJSON(data []byte) error {
	return json.Unmarshal(data, g)
}
