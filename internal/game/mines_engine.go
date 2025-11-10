package game

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	MINES_GRID_SIZE        = 25 // 5x5 grid
	MINES_MIN_COUNT        = 1
	MINES_MAX_COUNT        = 24
	REDIS_KEY_MINES_GAME   = "mines:game:"
	REDIS_KEY_MINES_BALANCE = "mines:balance:"
)

type MinesGameState struct {
	GameID       string    `json:"game_id"`
	UserID       string    `json:"user_id"`
	BetAmount    float64   `json:"bet_amount"`
	MineCount    int       `json:"mine_count"`
	ServerSeed   string    `json:"-"` // Hidden until game ends
	ClientSeed   string    `json:"client_seed"`
	Nonce        int       `json:"nonce"`
	MinePositions []int    `json:"-"` // Hidden until game ends
	RevealedTiles []int    `json:"revealed_tiles"`
	CurrentPayout float64  `json:"current_payout"`
	Status       string    `json:"status"` // ACTIVE, CASHED_OUT, BUSTED
	CreatedAt    time.Time `json:"created_at"`
	EndedAt      time.Time `json:"ended_at,omitempty"`
}

type MinesBetRequest struct {
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	MineCount int     `json:"mine_count"`
}

type MinesBetResponse struct {
	Success       bool    `json:"success"`
	Message       string  `json:"message"`
	GameID        string  `json:"game_id,omitempty"`
	Balance       float64 `json:"balance,omitempty"`
	CurrentPayout float64 `json:"current_payout"`
}

type MinesClickRequest struct {
	UserID string `json:"user_id"`
	GameID string `json:"game_id"`
	TileID int    `json:"tile_id"`
}

type MinesClickResponse struct {
	Success       bool    `json:"success"`
	Message       string  `json:"message"`
	TileID        int     `json:"tile_id"`
	IsMine        bool    `json:"is_mine"`
	CurrentPayout float64 `json:"current_payout"`
	GameStatus    string  `json:"game_status"`
	Balance       float64 `json:"balance,omitempty"`
}

type MinesCashoutRequest struct {
	UserID string `json:"user_id"`
	GameID string `json:"game_id"`
}
type MinesCashoutResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Payout  float64 `json:"payout"`
	Balance float64 `json:"balance"`
}

type MinesEngine struct {
	redisClient *redis.Client
	hub         *Hub
	ctx         context.Context
	nonce       int
}

func NewMinesEngine(redisClient *redis.Client, hub *Hub) *MinesEngine {
	return &MinesEngine{
		redisClient: redisClient,
		hub:         hub,
		ctx:         context.Background(),
		nonce:       0,
	}
}

func (m *MinesEngine) GetType() GameType {
	return GameTypeMines
}
func (m *MinesEngine) Start(ctx context.Context) error {
	m.ctx = ctx
	log.Println("[MINES] Engine started")
	return nil
}

func (m *MinesEngine) Stop() error {
	log.Println("[MINES] Engine stopped")
	return nil
}
func (m *MinesEngine) GetState() interface{} {
	return map[string]string{"status": "ready"}
}
func (m *MinesEngine) PlaceBet(ctx context.Context, req interface{}) (interface{}, error) {
	betReq, ok := req.(MinesBetRequest)
	if !ok {
		return nil, errors.New("invalid request type")
	}

	if betReq.MineCount < MINES_MIN_COUNT || betReq.MineCount > MINES_MAX_COUNT {
		return MinesBetResponse{
			Success: false,
			Message: fmt.Sprintf("Mine count must be between %d and %d", MINES_MIN_COUNT, MINES_MAX_COUNT),
		}, nil
	}

	if betReq.Amount < MIN_BET_AMOUNT || betReq.Amount > MAX_BET_AMOUNT {
		return MinesBetResponse{
			Success: false,
			Message: fmt.Sprintf("Bet must be between %.2f and %.2f", MIN_BET_AMOUNT, MAX_BET_AMOUNT),
		}, nil
	}

	balanceKey := REDIS_KEY_USER_BALANCE + betReq.UserID
	balance, err := m.redisClient.Get(ctx, balanceKey).Float64()
	if err != nil || balance < betReq.Amount {
		return MinesBetResponse{
			Success: false,
			Message: "Insufficient balance",
			Balance: balance,
		}, nil
	}

	newBalance, err := m.redisClient.IncrByFloat(ctx, balanceKey, -betReq.Amount).Result()
	if err != nil || newBalance < 0 {
		m.redisClient.IncrByFloat(ctx, balanceKey, betReq.Amount) // Rollback
		return MinesBetResponse{
			Success: false,
			Message: "Transaction failed",
		}, nil
	}

	// Generate provably fair mine positions
	m.nonce++
	serverSeed := GenerateSeed()
	clientSeed := GenerateSeed()
	minePositions := m.generateMinePositions(serverSeed, clientSeed, m.nonce, betReq.MineCount)

	// Create game state
	gameID := fmt.Sprintf("MINES-%s-%d", betReq.UserID, time.Now().UnixNano())
	gameState := MinesGameState{
		GameID:        gameID,
		UserID:        betReq.UserID,
		BetAmount:     betReq.Amount,
		MineCount:     betReq.MineCount,
		ServerSeed:    serverSeed,
		ClientSeed:    clientSeed,
		Nonce:         m.nonce,
		MinePositions: minePositions,
		RevealedTiles: []int{},
		CurrentPayout: betReq.Amount,
		Status:        "ACTIVE",
		CreatedAt:     time.Now(),
	}

	// Store game state in Redis
	gameKey := REDIS_KEY_MINES_GAME + gameID
	gameJSON, _ := json.Marshal(gameState)
	m.redisClient.Set(ctx, gameKey, gameJSON, 1*time.Hour)

	log.Printf("[MINES] Game %s started for user %s with %d mines", gameID, betReq.UserID, betReq.MineCount)

	return MinesBetResponse{
		Success:       true,
		Message:       "Game started",
		GameID:        gameID,
		Balance:       newBalance,
		CurrentPayout: betReq.Amount,
	}, nil
}

func (m *MinesEngine) ProcessAction(ctx context.Context, action string, req interface{}) (interface{}, error) {
	switch action {
	case "click":
		return m.handleTileClick(ctx, req)
	case "cashout":
		return m.handleCashout(ctx, req)
	default:
		return nil, errors.New("unknown action")
	}
}

// handleTileClick processes a tile click
func (m *MinesEngine) handleTileClick(ctx context.Context, req interface{}) (interface{}, error) {
	clickReq, ok := req.(MinesClickRequest)
	if !ok {
		return nil, errors.New("invalid request type")
	}

	// Load game state
	gameKey := REDIS_KEY_MINES_GAME + clickReq.GameID
	gameJSON, err := m.redisClient.Get(ctx, gameKey).Result()
	if err != nil {
		return MinesClickResponse{
			Success: false,
			Message: "Game not found",
		}, nil
	}

	var gameState MinesGameState
	json.Unmarshal([]byte(gameJSON), &gameState)

	if gameState.Status != "ACTIVE" {
		return MinesClickResponse{
			Success: false,
			Message: "Game is not active",
		}, nil
	}

	// Validate tile ID
	if clickReq.TileID < 0 || clickReq.TileID >= MINES_GRID_SIZE {
		return MinesClickResponse{
			Success: false,
			Message: "Invalid tile ID",
		}, nil
	}

	// Check if tile already revealed
	for _, revealed := range gameState.RevealedTiles {
		if revealed == clickReq.TileID {
			return MinesClickResponse{
				Success: false,
				Message: "Tile already revealed",
			}, nil
		}
	}

	// Check if tile is a mine
	isMine := false
	for _, minePos := range gameState.MinePositions {
		if minePos == clickReq.TileID {
			isMine = true
			break
		}
	}

	if isMine {
		// Player hit a mine - game over
		gameState.Status = "BUSTED"
		gameState.EndedAt = time.Now()
		gameState.CurrentPayout = 0

		// Update game state
		gameJSON, _ := json.Marshal(gameState)
		m.redisClient.Set(ctx, gameKey, gameJSON, 1*time.Hour)

		log.Printf("[MINES] User %s hit a mine at tile %d", clickReq.UserID, clickReq.TileID)

		return MinesClickResponse{
			Success:       true,
			Message:       "You hit a mine!",
			TileID:        clickReq.TileID,
			IsMine:        true,
			CurrentPayout: 0,
			GameStatus:    "BUSTED",
		}, nil
	}

	// Safe tile - update payout
	gameState.RevealedTiles = append(gameState.RevealedTiles, clickReq.TileID)
	gameState.CurrentPayout = m.calculatePayout(gameState.BetAmount, gameState.MineCount, len(gameState.RevealedTiles))

	// Update game state
	updatedGameJSON, _ := json.Marshal(gameState)
	m.redisClient.Set(ctx, gameKey, string(updatedGameJSON), 1*time.Hour)

	log.Printf("[MINES] User %s revealed safe tile %d, payout: %.2f", clickReq.UserID, clickReq.TileID, gameState.CurrentPayout)

	return MinesClickResponse{
		Success:       true,
		Message:       "Safe tile!",
		TileID:        clickReq.TileID,
		IsMine:        false,
		CurrentPayout: gameState.CurrentPayout,
		GameStatus:    "ACTIVE",
	}, nil
}

// handleCashout processes a cashout request
func (m *MinesEngine) handleCashout(ctx context.Context, req interface{}) (interface{}, error) {
	cashoutReq, ok := req.(MinesCashoutRequest)
	if !ok {
		return nil, errors.New("invalid request type")
	}

	// Load game state
	gameKey := REDIS_KEY_MINES_GAME + cashoutReq.GameID
	gameJSON, err := m.redisClient.Get(ctx, gameKey).Result()
	if err != nil {
		return MinesCashoutResponse{
			Success: false,
			Message: "Game not found",
		}, nil
	}

	var gameState MinesGameState
	json.Unmarshal([]byte(gameJSON), &gameState)

	// Validate game status
	if gameState.Status != "ACTIVE" {
		return MinesCashoutResponse{
			Success: false,
			Message: "Game is not active",
		}, nil
	}

	// Must have revealed at least one tile
	if len(gameState.RevealedTiles) == 0 {
		return MinesCashoutResponse{
			Success: false,
			Message: "Must reveal at least one tile before cashing out",
		}, nil
	}

	// Update game status
	gameState.Status = "CASHED_OUT"
	gameState.EndedAt = time.Now()

	// Credit user balance
	balanceKey := REDIS_KEY_USER_BALANCE + cashoutReq.UserID
	newBalance, err := m.redisClient.IncrByFloat(ctx, balanceKey, gameState.CurrentPayout).Result()
	if err != nil {
		return MinesCashoutResponse{
			Success: false,
			Message: "Failed to credit balance",
		}, nil
	}

	// Update game state
	gameJSONBytes, _ := json.Marshal(gameState)
	m.redisClient.Set(ctx, gameKey, string(gameJSONBytes), 1*time.Hour)

	log.Printf("[MINES] User %s cashed out for %.2f", cashoutReq.UserID, gameState.CurrentPayout)

	return MinesCashoutResponse{
		Success: true,
		Message: "Cashed out successfully",
		Payout:  gameState.CurrentPayout,
		Balance: newBalance,
	}, nil
}

// generateMinePositions generates mine positions using provably fair algorithm
func (m *MinesEngine) generateMinePositions(serverSeed, clientSeed string, nonce, mineCount int) []int {
	positions := make([]int, 0, mineCount)
	used := make(map[int]bool)

	// Use the hash to generate mine positions
	for i := 0; len(positions) < mineCount && i < 100; i++ {
		// Create a new hash for each position
		posHash := hmac.New(sha256.New, []byte(serverSeed))
		posHash.Write([]byte(fmt.Sprintf("%s:%d:%d", clientSeed, nonce, i)))
		posHashBytes := posHash.Sum(nil)
		posHashHex := hex.EncodeToString(posHashBytes)

		// Take first 8 hex characters
		hexValue := posHashHex[:8]
		bigInt := new(big.Int)
		bigInt.SetString(hexValue, 16)

		// Map to grid position
		position := int(bigInt.Uint64() % uint64(MINES_GRID_SIZE))

		if !used[position] {
			positions = append(positions, position)
			used[position] = true
		}
	}

	return positions
}

// calculatePayout calculates the current payout based on revealed tiles
func (m *MinesEngine) calculatePayout(betAmount float64, mineCount, revealedCount int) float64 {
	if revealedCount == 0 {
		return betAmount
	}

	// Calculate multiplier based on probability
	// Formula: multiplier = (totalTiles / safeTiles) ^ revealedCount * houseEdge
	totalTiles := float64(MINES_GRID_SIZE)
	safeTiles := totalTiles - float64(mineCount)
	houseEdge := 0.97 // 3% house edge

	multiplier := 1.0
	for i := 0; i < revealedCount; i++ {
		multiplier *= (totalTiles - float64(i)) / (safeTiles - float64(i))
	}

	multiplier *= houseEdge

	payout := betAmount * multiplier
	return float64(int(payout*100)) / 100.0 // Round to 2 decimal places
}
