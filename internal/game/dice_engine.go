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
	REDIS_KEY_DICE_GAME = "dice:game:"
	DICE_MIN_VALUE      = 0.00
	DICE_MAX_VALUE      = 100.00
)

// DiceGameState represents a completed Dice game
type DiceGameState struct {
	GameID     string    `json:"game_id"`
	UserID     string    `json:"user_id"`
	BetAmount  float64   `json:"bet_amount"`
	Target     float64   `json:"target"`
	IsOver     bool      `json:"is_over"` // true = roll over, false = roll under
	ServerSeed string    `json:"server_seed"`
	ClientSeed string    `json:"client_seed"`
	Nonce      int       `json:"nonce"`
	RollResult float64   `json:"roll_result"`
	Win        bool      `json:"win"`
	Multiplier float64   `json:"multiplier"`
	Payout     float64   `json:"payout"`
	CreatedAt  time.Time `json:"created_at"`
}

// DiceRollRequest represents a dice roll request
type DiceRollRequest struct {
	UserID string  `json:"user_id"`
	Amount float64 `json:"amount"`
	Target float64 `json:"target"`
	IsOver bool    `json:"is_over"`
}

// DiceRollResponse represents the response to a dice roll
type DiceRollResponse struct {
	Success    bool    `json:"success"`
	Message    string  `json:"message"`
	GameID     string  `json:"game_id,omitempty"`
	RollResult float64 `json:"roll_result,omitempty"`
	Win        bool    `json:"win,omitempty"`
	Multiplier float64 `json:"multiplier,omitempty"`
	Payout     float64 `json:"payout,omitempty"`
	Balance    float64 `json:"balance,omitempty"`
	ServerSeed string  `json:"server_seed,omitempty"`
	ClientSeed string  `json:"client_seed,omitempty"`
	Nonce      int     `json:"nonce,omitempty"`
}

// DiceEngine implements the GameEngine interface for Dice game
type DiceEngine struct {
	redisClient *redis.Client
	hub         *Hub
	ctx         context.Context
	nonce       int
}

// NewDiceEngine creates a new Dice game engine
func NewDiceEngine(redisClient *redis.Client, hub *Hub) *DiceEngine {
	return &DiceEngine{
		redisClient: redisClient,
		hub:         hub,
		ctx:         context.Background(),
		nonce:       0,
	}
}

// GetType returns the game type
func (d *DiceEngine) GetType() GameType {
	return GameTypeDice
}

// Start initializes the Dice engine
func (d *DiceEngine) Start(ctx context.Context) error {
	d.ctx = ctx
	log.Println("[DICE] Engine started")
	return nil
}

// Stop gracefully stops the Dice engine
func (d *DiceEngine) Stop() error {
	log.Println("[DICE] Engine stopped")
	return nil
}

// GetState returns the current game state (not applicable for Dice)
func (d *DiceEngine) GetState() interface{} {
	return map[string]string{"status": "ready"}
}

// PlaceBet handles a dice roll (instant result)
func (d *DiceEngine) PlaceBet(ctx context.Context, req interface{}) (interface{}, error) {
	rollReq, ok := req.(DiceRollRequest)
	if !ok {
		return nil, errors.New("invalid request type")
	}

	// Validate bet amount
	if rollReq.Amount < MIN_BET_AMOUNT || rollReq.Amount > MAX_BET_AMOUNT {
		return DiceRollResponse{
			Success: false,
			Message: fmt.Sprintf("Bet must be between %.2f and %.2f", MIN_BET_AMOUNT, MAX_BET_AMOUNT),
		}, nil
	}

	// Validate target
	if rollReq.Target < DICE_MIN_VALUE || rollReq.Target > DICE_MAX_VALUE {
		return DiceRollResponse{
			Success: false,
			Message: fmt.Sprintf("Target must be between %.2f and %.2f", DICE_MIN_VALUE, DICE_MAX_VALUE),
		}, nil
	}

	// Validate target range (must allow for possible win)
	if rollReq.IsOver && rollReq.Target >= 99.00 {
		return DiceRollResponse{
			Success: false,
			Message: "Target too high for 'over' bet",
		}, nil
	}
	if !rollReq.IsOver && rollReq.Target <= 1.00 {
		return DiceRollResponse{
			Success: false,
			Message: "Target too low for 'under' bet",
		}, nil
	}

	// Check user balance
	balanceKey := REDIS_KEY_USER_BALANCE + rollReq.UserID
	balance, err := d.redisClient.Get(ctx, balanceKey).Float64()
	if err != nil || balance < rollReq.Amount {
		return DiceRollResponse{
			Success: false,
			Message: "Insufficient balance",
			Balance: balance,
		}, nil
	}

	// Deduct balance
	newBalance, err := d.redisClient.IncrByFloat(ctx, balanceKey, -rollReq.Amount).Result()
	if err != nil || newBalance < 0 {
		d.redisClient.IncrByFloat(ctx, balanceKey, rollReq.Amount) // Rollback
		return DiceRollResponse{
			Success: false,
			Message: "Transaction failed",
		}, nil
	}

	// Generate provably fair result
	d.nonce++
	serverSeed := GenerateSeed()
	clientSeed := GenerateSeed()
	rollResult := d.generateRoll(serverSeed, clientSeed, d.nonce)

	// Determine win
	win := false
	if rollReq.IsOver {
		win = rollResult > rollReq.Target
	} else {
		win = rollResult < rollReq.Target
	}

	// Calculate multiplier and payout
	multiplier := d.calculateMultiplier(rollReq.Target, rollReq.IsOver)
	payout := 0.0
	if win {
		payout = rollReq.Amount * multiplier
	}

	// Credit payout if won
	finalBalance := newBalance
	if win {
		finalBalance, err = d.redisClient.IncrByFloat(ctx, balanceKey, payout).Result()
		if err != nil {
			return DiceRollResponse{
				Success: false,
				Message: "Failed to credit payout",
			}, nil
		}
	}

	// Create game state
	gameID := fmt.Sprintf("DICE-%s-%d", rollReq.UserID, time.Now().UnixNano())
	gameState := DiceGameState{
		GameID:     gameID,
		UserID:     rollReq.UserID,
		BetAmount:  rollReq.Amount,
		Target:     rollReq.Target,
		IsOver:     rollReq.IsOver,
		ServerSeed: serverSeed,
		ClientSeed: clientSeed,
		Nonce:      d.nonce,
		RollResult: rollResult,
		Win:        win,
		Multiplier: multiplier,
		Payout:     payout,
		CreatedAt:  time.Now(),
	}

	// Store game state in Redis
	gameKey := REDIS_KEY_DICE_GAME + gameID
	gameJSON, _ := json.Marshal(gameState)
	d.redisClient.Set(ctx, gameKey, string(gameJSON), 1*time.Hour)

	winStatus := "lost"
	if win {
		winStatus = "won"
	}
	log.Printf("[DICE] User %s rolled %.2f (%s %.2f), %s, payout %.2f",
		rollReq.UserID, rollResult, map[bool]string{true: "over", false: "under"}[rollReq.IsOver],
		rollReq.Target, winStatus, payout)

	return DiceRollResponse{
		Success:    true,
		Message:    "Dice rolled successfully",
		GameID:     gameID,
		RollResult: rollResult,
		Win:        win,
		Multiplier: multiplier,
		Payout:     payout,
		Balance:    finalBalance,
		ServerSeed: serverSeed,
		ClientSeed: clientSeed,
		Nonce:      d.nonce,
	}, nil
}

func (d *DiceEngine) ProcessAction(ctx context.Context, action string, req interface{}) (interface{}, error) {
	return nil, errors.New("no actions available for Dice")
}

// generateRoll generates a dice roll result using provably fair algorithm
func (d *DiceEngine) generateRoll(serverSeed, clientSeed string, nonce int) float64 {
	data := fmt.Sprintf("%s:%d", clientSeed, nonce)
	h := hmac.New(sha256.New, []byte(serverSeed))
	h.Write([]byte(data))
	hashBytes := h.Sum(nil)
	hashHex := hex.EncodeToString(hashBytes)

	// Take first 16 hex characters (64 bits)
	hexValue := hashHex[:16]
	bigInt := new(big.Int)
	bigInt.SetString(hexValue, 16)

	// Convert to float between 0 and 100
	const MAX_VALUE_F64 = 18446744073709551616.0
	result := (float64(bigInt.Uint64()) / MAX_VALUE_F64) * 100.0

	// Round to 2 decimal places
	return float64(int(result*100)) / 100.0
}

// calculateMultiplier calculates the payout multiplier based on win probability
func (d *DiceEngine) calculateMultiplier(target float64, isOver bool) float64 {
	// Calculate win probability
	var winChance float64
	if isOver {
		winChance = (100.0 - target) / 100.0
	} else {
		winChance = target / 100.0
	}

	// Prevent division by zero
	if winChance <= 0.01 {
		winChance = 0.01
	}

	// House edge: 1%
	houseEdge := 0.99

	// Multiplier = (1 / winChance) * houseEdge
	multiplier := (1.0 / winChance) * houseEdge

	// Round to 2 decimal places
	return float64(int(multiplier*100)) / 100.0
}
