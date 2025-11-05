package game

import (
	"time"
)

type BetRequest struct {
	UserID       string  `json:"user_id"`
	Amount       float64 `json:"amount"`
	AutoCashout  float64 `json:"auto_cashout,omitempty"`
	RoundID      string  `json:"round_id"`
	ResponseChan chan BetResponse `json:"-"`
}

type BetResponse struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	BetID   string  `json:"bet_id,omitempty"`
	Balance float64 `json:"balance,omitempty"`
}

type CashoutRequest struct {
	UserID       string `json:"user_id"`
	BetID        string `json:"bet_id"`
	RoundID      string `json:"round_id"`
	ResponseChan chan CashoutResponse `json:"-"`
}

type CashoutResponse struct {
	Success    bool    `json:"success"`
	Message    string  `json:"message"`
	Multiplier float64 `json:"multiplier,omitempty"`
	Payout     float64 `json:"payout,omitempty"`
	Balance    float64 `json:"balance,omitempty"`
}

type RoundState struct {
	RoundID           string    `json:"round_id"`
	ServerSeed        string    `json:"-"` // Never expose until reveal
	HashCommitment    string    `json:"hash_commitment"`
	ClientSeed        string    `json:"client_seed"`
	CrashMultiplier   float64   `json:"-"` // Hidden until crash
	CurrentMultiplier float64   `json:"current_multiplier"`
	Status            string    `json:"status"` // BETTING, RUNNING, CRASHED
	StartTime         time.Time `json:"start_time"`
	CrashTime         time.Time `json:"crash_time,omitempty"`
	Nonce             int       `json:"nonce"`
}

type ActiveBet struct {
	BetID       string    `json:"bet_id"`
	UserID      string    `json:"user_id"`
	Amount      float64   `json:"amount"`
	AutoCashout float64   `json:"auto_cashout"`
	PlacedAt    time.Time `json:"placed_at"`
	CashedOut   bool      `json:"cashed_out"`
}

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
}

type BetPlacedMessage struct {
	UserID string  `json:"user_id"`
	Amount float64 `json:"amount"`
	BetID  string  `json:"bet_id"`
}

type CashoutMessage struct {
	UserID     string  `json:"user_id"`
	BetID      string  `json:"bet_id"`
	Multiplier float64 `json:"multiplier"`
	Payout     float64 `json:"payout"`
}
