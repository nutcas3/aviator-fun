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
	REDIS_KEY_PLINKO_GAME = "plinko:game:"
)

// PlinkoRisk represents the risk level
type PlinkoRisk string

const (
	PlinkoRiskLow    PlinkoRisk = "low"
	PlinkoRiskMedium PlinkoRisk = "medium"
	PlinkoRiskHigh   PlinkoRisk = "high"
)

// Plinko multipliers for each risk level (16 rows)
var plinkoMultipliers = map[PlinkoRisk][]float64{
	PlinkoRiskLow: {
		16.0, 9.0, 2.0, 1.4, 1.4, 1.2, 1.1, 1.0,
		0.5, 1.0, 1.1, 1.2, 1.4, 1.4, 2.0, 9.0, 16.0,
	},
	PlinkoRiskMedium: {
		110.0, 41.0, 10.0, 5.0, 3.0, 1.5, 1.0, 0.5,
		0.3, 0.5, 1.0, 1.5, 3.0, 5.0, 10.0, 41.0, 110.0,
	},
	PlinkoRiskHigh: {
		1000.0, 130.0, 26.0, 9.0, 4.0, 2.0, 0.2, 0.2,
		0.2, 0.2, 0.2, 2.0, 4.0, 9.0, 26.0, 130.0, 1000.0,
	},
}

// PlinkoGameState represents a completed Plinko game
type PlinkoGameState struct {
	GameID     string     `json:"game_id"`
	UserID     string     `json:"user_id"`
	BetAmount  float64    `json:"bet_amount"`
	Risk       PlinkoRisk `json:"risk"`
	Rows       int        `json:"rows"`
	ServerSeed string     `json:"server_seed"`
	ClientSeed string     `json:"client_seed"`
	Nonce      int        `json:"nonce"`
	Path       []int      `json:"path"`        // 0 = left, 1 = right
	LandingSlot int       `json:"landing_slot"`
	Multiplier float64    `json:"multiplier"`
	Payout     float64    `json:"payout"`
	CreatedAt  time.Time  `json:"created_at"`
}

// PlinkoDropRequest represents a ball drop request
type PlinkoDropRequest struct {
	UserID string     `json:"user_id"`
	Amount float64    `json:"amount"`
	Risk   PlinkoRisk `json:"risk"`
	Rows   int        `json:"rows"`
}

// PlinkoDropResponse represents the response to a ball drop
type PlinkoDropResponse struct {
	Success     bool       `json:"success"`
	Message     string     `json:"message"`
	GameID      string     `json:"game_id,omitempty"`
	Path        []int      `json:"path,omitempty"`
	LandingSlot int        `json:"landing_slot,omitempty"`
	Multiplier  float64    `json:"multiplier,omitempty"`
	Payout      float64    `json:"payout,omitempty"`
	Balance     float64    `json:"balance,omitempty"`
	ServerSeed  string     `json:"server_seed,omitempty"`
	ClientSeed  string     `json:"client_seed,omitempty"`
	Nonce       int        `json:"nonce,omitempty"`
}

// PlinkoEngine implements the GameEngine interface for Plinko game
type PlinkoEngine struct {
	redisClient *redis.Client
	hub         *Hub
	ctx         context.Context
	nonce       int
}

// NewPlinkoEngine creates a new Plinko game engine
func NewPlinkoEngine(redisClient *redis.Client, hub *Hub) *PlinkoEngine {
	return &PlinkoEngine{
		redisClient: redisClient,
		hub:         hub,
		ctx:         context.Background(),
		nonce:       0,
	}
}

// GetType returns the game type
func (p *PlinkoEngine) GetType() GameType {
	return GameTypePlinko
}

// Start initializes the Plinko engine
func (p *PlinkoEngine) Start(ctx context.Context) error {
	p.ctx = ctx
	log.Println("[PLINKO] Engine started")
	return nil
}

// Stop gracefully stops the Plinko engine
func (p *PlinkoEngine) Stop() error {
	log.Println("[PLINKO] Engine stopped")
	return nil
}

// GetState returns the current game state (not applicable for Plinko)
func (p *PlinkoEngine) GetState() interface{} {
	return map[string]string{"status": "ready"}
}

// PlaceBet handles a ball drop for Plinko (instant result)
func (p *PlinkoEngine) PlaceBet(ctx context.Context, req interface{}) (interface{}, error) {
	dropReq, ok := req.(PlinkoDropRequest)
	if !ok {
		return nil, errors.New("invalid request type")
	}

	// Validate bet amount
	if dropReq.Amount < MIN_BET_AMOUNT || dropReq.Amount > MAX_BET_AMOUNT {
		return PlinkoDropResponse{
			Success: false,
			Message: fmt.Sprintf("Bet must be between %.2f and %.2f", MIN_BET_AMOUNT, MAX_BET_AMOUNT),
		}, nil
	}

	// Validate rows (8, 12, or 16)
	if dropReq.Rows != 8 && dropReq.Rows != 12 && dropReq.Rows != 16 {
		return PlinkoDropResponse{
			Success: false,
			Message: "Rows must be 8, 12, or 16",
		}, nil
	}

	// Validate risk level
	if dropReq.Risk != PlinkoRiskLow && dropReq.Risk != PlinkoRiskMedium && dropReq.Risk != PlinkoRiskHigh {
		return PlinkoDropResponse{
			Success: false,
			Message: "Risk must be low, medium, or high",
		}, nil
	}

	// Check user balance
	balanceKey := REDIS_KEY_USER_BALANCE + dropReq.UserID
	balance, err := p.redisClient.Get(ctx, balanceKey).Float64()
	if err != nil || balance < dropReq.Amount {
		return PlinkoDropResponse{
			Success: false,
			Message: "Insufficient balance",
			Balance: balance,
		}, nil
	}

	// Deduct balance
	newBalance, err := p.redisClient.IncrByFloat(ctx, balanceKey, -dropReq.Amount).Result()
	if err != nil || newBalance < 0 {
		p.redisClient.IncrByFloat(ctx, balanceKey, dropReq.Amount) // Rollback
		return PlinkoDropResponse{
			Success: false,
			Message: "Transaction failed",
		}, nil
	}

	// Generate provably fair result
	p.nonce++
	serverSeed := GenerateSeed()
	clientSeed := GenerateSeed()
	path, landingSlot := p.generatePath(serverSeed, clientSeed, p.nonce, dropReq.Rows)
	multiplier := p.getMultiplier(dropReq.Risk, landingSlot, dropReq.Rows)
	payout := dropReq.Amount * multiplier

	// Credit payout
	finalBalance, err := p.redisClient.IncrByFloat(ctx, balanceKey, payout).Result()
	if err != nil {
		return PlinkoDropResponse{
			Success: false,
			Message: "Failed to credit payout",
		}, nil
	}

	// Create game state
	gameID := fmt.Sprintf("PLINKO-%s-%d", dropReq.UserID, time.Now().UnixNano())
	gameState := PlinkoGameState{
		GameID:      gameID,
		UserID:      dropReq.UserID,
		BetAmount:   dropReq.Amount,
		Risk:        dropReq.Risk,
		Rows:        dropReq.Rows,
		ServerSeed:  serverSeed,
		ClientSeed:  clientSeed,
		Nonce:       p.nonce,
		Path:        path,
		LandingSlot: landingSlot,
		Multiplier:  multiplier,
		Payout:      payout,
		CreatedAt:   time.Now(),
	}

	// Store game state in Redis
	gameKey := REDIS_KEY_PLINKO_GAME + gameID
	gameJSON, _ := json.Marshal(gameState)
	p.redisClient.Set(ctx, gameKey, string(gameJSON), 1*time.Hour)

	log.Printf("[PLINKO] User %s dropped ball, landed at slot %d, multiplier %.2fx, payout %.2f",
		dropReq.UserID, landingSlot, multiplier, payout)

	return PlinkoDropResponse{
		Success:     true,
		Message:     "Ball dropped successfully",
		GameID:      gameID,
		Path:        path,
		LandingSlot: landingSlot,
		Multiplier:  multiplier,
		Payout:      payout,
		Balance:     finalBalance,
		ServerSeed:  serverSeed,
		ClientSeed:  clientSeed,
		Nonce:       p.nonce,
	}, nil
}

// ProcessAction handles game-specific actions (not applicable for Plinko)
func (p *PlinkoEngine) ProcessAction(ctx context.Context, action string, req interface{}) (interface{}, error) {
	return nil, errors.New("no actions available for Plinko")
}

// generatePath generates the ball's path using provably fair algorithm
func (p *PlinkoEngine) generatePath(serverSeed, clientSeed string, nonce, rows int) ([]int, int) {
	path := make([]int, rows)
	position := 0

	for i := 0; i < rows; i++ {
		// Generate hash for this step
		data := fmt.Sprintf("%s:%d:%d", clientSeed, nonce, i)
		h := hmac.New(sha256.New, []byte(serverSeed))
		h.Write([]byte(data))
		hashBytes := h.Sum(nil)
		hashHex := hex.EncodeToString(hashBytes)

		// Take first 8 hex characters
		hexValue := hashHex[:8]
		bigInt := new(big.Int)
		bigInt.SetString(hexValue, 16)

		// Determine direction: 0 = left, 1 = right
		direction := int(bigInt.Uint64() % 2)
		path[i] = direction

		// Update position
		if direction == 1 {
			position++
		}
	}

	return path, position
}

// getMultiplier returns the multiplier for a given landing slot
func (p *PlinkoEngine) getMultiplier(risk PlinkoRisk, landingSlot, rows int) float64 {
	multipliers, exists := plinkoMultipliers[risk]
	if !exists {
		return 1.0
	}

	// Scale multipliers based on rows (16 rows is the base)
	scaleFactor := float64(rows) / 16.0

	// Ensure landing slot is within bounds
	if landingSlot < 0 {
		landingSlot = 0
	}
	if landingSlot >= len(multipliers) {
		landingSlot = len(multipliers) - 1
	}

	baseMultiplier := multipliers[landingSlot]
	
	// Apply scaling for different row counts
	if rows < 16 {
		// For fewer rows, reduce extreme multipliers
		if baseMultiplier > 10.0 {
			baseMultiplier = 10.0 + (baseMultiplier-10.0)*scaleFactor
		}
	}

	return baseMultiplier
}
