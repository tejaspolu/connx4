package main

import (
	"math"
	"math/rand"
	"time"
)

const (
	DifficultyEasy   = "Easy"
	DifficultyMedium = "Medium"
	DifficultyHard   = "Hard"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GetAIMove(g *Game, difficulty string, aiPlayer int) int {
	switch difficulty {
	case DifficultyEasy:
		return getRandomMove(g)
	case DifficultyMedium:
		return getMediumMove(g, aiPlayer)
	case DifficultyHard:
		return getHardMove(g, aiPlayer)
	default:
		return getRandomMove(g)
	}
}

func getRandomMove(g *Game) int {
	validMoves := getValidMoves(g)
	if len(validMoves) == 0 {
		return -1
	}
	return validMoves[rand.Intn(len(validMoves))]
}

func getMediumMove(g *Game, aiPlayer int) int {
	opponent := localOtherPlayer(aiPlayer)
	validMoves := getValidMoves(g)
	if len(validMoves) == 0 {
		return -1
	}

	for _, col := range validMoves {
		sim := cloneGame(g)
		sim.CurrentTurn = aiPlayer
		sim.DropPiece(col)
		if sim.IsOver && sim.Winner == aiPlayer {
			return col
		}
	}

	for _, col := range validMoves {
		sim := cloneGame(g)
		sim.CurrentTurn = opponent
		sim.DropPiece(col)
		if sim.IsOver && sim.Winner == opponent {
			return col
		}
	}

	return getRandomMove(g)
}

func getHardMove(g *Game, aiPlayer int) int {
	validMoves := getValidMoves(g)
	if len(validMoves) == 0 {
		return -1
	}

	opponent := localOtherPlayer(aiPlayer)
	bestScore := math.Inf(-1)
	bestCol := validMoves[0]

	ordered := centerFirstOrder()

	for _, col := range ordered {
		if !isValidMove(g, col) {
			continue
		}
		sim := cloneGame(g)
		sim.CurrentTurn = aiPlayer
		sim.DropPiece(col)
		score := minimax(sim, 4, false, aiPlayer, opponent, math.Inf(-1), math.Inf(1))
		if score > bestScore {
			bestScore = score
			bestCol = col
		}
	}
	return bestCol
}

func centerFirstOrder() []int {
	return []int{3, 2, 4, 1, 5, 0, 6}
}

func getValidMoves(g *Game) []int {
	var moves []int
	for col := 0; col < 7; col++ {
		if g.Board[0][col] == 0 {
			moves = append(moves, col)
		}
	}
	return moves
}

func isValidMove(g *Game, col int) bool {
	return col >= 0 && col < 7 && g.Board[0][col] == 0
}

func cloneGame(g *Game) *Game {
	copyGame := *g
	return &copyGame
}

func localOtherPlayer(player int) int {
	if player == 1 {
		return 2
	}
	return 1
}

func minimax(g *Game, depth int, maximizing bool, aiPlayer, opponent int, alpha, beta float64) float64 {
	validMoves := getValidMoves(g)

	if depth == 0 || g.IsOver || len(validMoves) == 0 {
		return evaluateBoard(g, aiPlayer, opponent)
	}

	if maximizing {
		maxEval := math.Inf(-1)
		for _, col := range validMoves {
			sim := cloneGame(g)
			sim.CurrentTurn = aiPlayer
			sim.DropPiece(col)
			eval := minimax(sim, depth-1, false, aiPlayer, opponent, alpha, beta)
			if eval > maxEval {
				maxEval = eval
			}
			if maxEval > alpha {
				alpha = maxEval
			}
			if beta <= alpha {
				break
			}
		}
		return maxEval
	}

	minEval := math.Inf(1)
	for _, col := range validMoves {
		sim := cloneGame(g)
		sim.CurrentTurn = opponent
		sim.DropPiece(col)
		eval := minimax(sim, depth-1, true, aiPlayer, opponent, alpha, beta)
		if eval < minEval {
			minEval = eval
		}
		if minEval < beta {
			beta = minEval
		}
		if beta <= alpha {
			break
		}
	}
	return minEval
}

func evaluateBoard(g *Game, aiPlayer, opponent int) float64 {
	if g.IsOver {
		if g.Winner == aiPlayer {
			return 100000
		} else if g.Winner == opponent {
			return -100000
		}
		return 0
	}

	score := 0.0

	centerCol := 3
	centerCount := 0
	for row := 0; row < 6; row++ {
		if g.Board[row][centerCol] == aiPlayer {
			centerCount++
		}
	}
	score += float64(centerCount * 6)

	for row := 0; row < 6; row++ {
		for col := 0; col < 4; col++ {
			window := []int{
				g.Board[row][col],
				g.Board[row][col+1],
				g.Board[row][col+2],
				g.Board[row][col+3],
			}
			score += evaluateWindow(window, aiPlayer, opponent)
		}
	}

	for col := 0; col < 7; col++ {
		for row := 0; row < 3; row++ {
			window := []int{
				g.Board[row][col],
				g.Board[row+1][col],
				g.Board[row+2][col],
				g.Board[row+3][col],
			}
			score += evaluateWindow(window, aiPlayer, opponent)
		}
	}

	for row := 0; row < 3; row++ {
		for col := 0; col < 4; col++ {
			window := []int{
				g.Board[row][col],
				g.Board[row+1][col+1],
				g.Board[row+2][col+2],
				g.Board[row+3][col+3],
			}
			score += evaluateWindow(window, aiPlayer, opponent)
		}
	}

	for row := 3; row < 6; row++ {
		for col := 0; col < 4; col++ {
			window := []int{
				g.Board[row][col],
				g.Board[row-1][col+1],
				g.Board[row-2][col+2],
				g.Board[row-3][col+3],
			}
			score += evaluateWindow(window, aiPlayer, opponent)
		}
	}

	return score
}

func evaluateWindow(window []int, aiPlayer, opponent int) float64 {
	aiCount := 0
	oppCount := 0
	emptyCount := 0

	for _, v := range window {
		switch v {
		case aiPlayer:
			aiCount++
		case opponent:
			oppCount++
		case 0:
			emptyCount++
		}
	}

	score := 0.0

	if aiCount == 4 {
		score += 1000
	} else if aiCount == 3 && emptyCount == 1 {
		score += 50
	} else if aiCount == 2 && emptyCount == 2 {
		score += 10
	}

	if oppCount == 3 && emptyCount == 1 {
		score -= 80
	} else if oppCount == 4 {
		score -= 1000
	}

	return score
}
