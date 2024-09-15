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

func (g *Game) FromJSON(data []byte) error {
	return json.Unmarshal(data, g)
}
