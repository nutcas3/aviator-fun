package game

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	TICK_INTERVAL  = 100 * time.Millisecond
	BETTING_TIME   = 5 * time.Second
	MAX_BET_AMOUNT = 10000.0
	MIN_BET_AMOUNT = 1.0
	CASHOUT_TIMEOUT = 500 * time.Millisecond

	REDIS_KEY_ROUND_PREFIX = "crash:round:"
	REDIS_KEY_ACTIVE_BETS  = "crash:bets:active:"
	REDIS_KEY_USER_BALANCE = "crash:balance:"
	REDIS_KEY_ROUND_LOCK   = "crash:lock:round"
)

type Manager struct {
	hub            *Hub
	redisClient    *redis.Client
	ctx            context.Context
	currentRound   *RoundState
	stateMutex     sync.RWMutex
	betChannel     chan BetRequest
	cashoutChannel chan CashoutRequest
	stopChan       chan struct{}
	nonce          int
}

func NewManager(hub *Hub, redisClient *redis.Client) *Manager {
	return &Manager{
		hub:            hub,
		redisClient:    redisClient,
		ctx:            context.Background(),
		betChannel:     make(chan BetRequest, 1000),
		cashoutChannel: make(chan CashoutRequest, 1000),
		stopChan:       make(chan struct{}),
		nonce:          0,
	}
}

func (m *Manager) Start() {
	go m.gameLoop()
}

func (m *Manager) Stop() {
	close(m.stopChan)
}

func (m *Manager) GetCurrentRound() *RoundState {
	m.stateMutex.RLock()
	defer m.stateMutex.RUnlock()
	if m.currentRound == nil {
		return nil
	}
	roundCopy := *m.currentRound
	return &roundCopy
}

func (m *Manager) PlaceBet(req BetRequest) BetResponse {
	respChan := make(chan BetResponse, 1)
	req.ResponseChan = respChan

	select {
	case m.betChannel <- req:
		select {
		case resp := <-respChan:
			return resp
		case <-time.After(5 * time.Second):
			return BetResponse{Success: false, Message: "Bet timeout"}
		}
	default:
		return BetResponse{Success: false, Message: "Bet queue full"}
	}
}

func (m *Manager) Cashout(req CashoutRequest) CashoutResponse {
	respChan := make(chan CashoutResponse, 1)
	req.ResponseChan = respChan

	select {
	case m.cashoutChannel <- req:
		select {
		case resp := <-respChan:
			return resp
		case <-time.After(CASHOUT_TIMEOUT):
			return CashoutResponse{Success: false, Message: "Cashout timeout"}
		}
	default:
		return CashoutResponse{Success: false, Message: "Cashout queue full"}
	}
}

func (m *Manager) gameLoop() {
	for {
		select {
		case <-m.stopChan:
			log.Println("[GAME] Game loop stopped")
			return
		default:
			m.runRound()
		}
	}
}

func (m *Manager) runRound() {
	m.nonce++

	serverSeed := GenerateSeed()
	commitment := HashCommitment(serverSeed)
	clientSeed := GenerateSeed() // In production, aggregate from player inputs
	crashPoint := HashAndMapToMultiplier(serverSeed, clientSeed, m.nonce)

	roundID := fmt.Sprintf("R%d-%d", time.Now().Unix(), m.nonce)

	m.stateMutex.Lock()
	m.currentRound = &RoundState{
		RoundID:           roundID,
		ServerSeed:        serverSeed,
		HashCommitment:    commitment,
		ClientSeed:        clientSeed,
		CrashMultiplier:   crashPoint,
		CurrentMultiplier: MIN_MULTIPLIER,
		Status:            "BETTING",
		StartTime:         time.Now(),
		Nonce:             m.nonce,
	}
	m.stateMutex.Unlock()

	m.storeRoundInRedis(m.currentRound)

	log.Printf("\n=== ROUND %s ===", roundID)
	log.Printf("[FAIR] Commitment: %s", commitment[:16]+"...")
	log.Printf("[FAIR] Crash Point: %.2fx (HIDDEN)", crashPoint)

	m.hub.Broadcast(map[string]interface{}{
		"type":       "round_start",
		"status":     "BETTING",
		"round_id":   roundID,
		"commitment": commitment,
		"time_left":  BETTING_TIME.Seconds(),
	})

	bettingTimer := time.NewTimer(BETTING_TIME)
	bettingLoop := true

	for bettingLoop {
		select {
		case <-bettingTimer.C:
			bettingLoop = false
		case bet := <-m.betChannel:
			m.processBet(bet)
		case <-m.stopChan:
			return
		}
	}

	m.stateMutex.Lock()
	m.currentRound.Status = "RUNNING"
	m.stateMutex.Unlock()

	m.hub.Broadcast(map[string]interface{}{
		"type":     "round_running",
		"status":   "RUNNING",
		"round_id": roundID,
	})

	ticker := time.NewTicker(TICK_INTERVAL)
	defer ticker.Stop()

	startTime := time.Now()
	activeBets := m.loadActiveBets(roundID)

	runningLoop := true
	for runningLoop {
		select {
		case <-ticker.C:
			m.stateMutex.Lock()

			elapsed := time.Since(startTime).Seconds()
			m.currentRound.CurrentMultiplier = calculateMultiplier(elapsed)
			currentMult := m.currentRound.CurrentMultiplier

			if currentMult >= m.currentRound.CrashMultiplier {
				m.currentRound.Status = "CRASHED"
				m.currentRound.CurrentMultiplier = m.currentRound.CrashMultiplier
				m.currentRound.CrashTime = time.Now()

				m.hub.Broadcast(map[string]interface{}{
					"type":        "crash",
					"multiplier":  m.currentRound.CrashMultiplier,
					"server_seed": m.currentRound.ServerSeed,
					"round_id":    roundID,
				})

				// Process remaining bets as losses
				m.processRoundEnd(roundID, activeBets)

				m.stateMutex.Unlock()
				runningLoop = false
				break
			}

			// Broadcast update
			m.hub.Broadcast(map[string]interface{}{
				"type":       "update",
				"multiplier": currentMult,
				"round_id":   roundID,
			})

			// Check auto-cashouts
			m.processAutoCashouts(roundID, currentMult, activeBets)

			m.stateMutex.Unlock()

		case cashout := <-m.cashoutChannel:
			m.processCashout(cashout)

		case <-m.stopChan:
			return
		}
	}

	log.Printf("=== ROUND %s ENDED at %.2fx ===\n", roundID, crashPoint)

	// Pause between rounds
	time.Sleep(3 * time.Second)
}

// calculateMultiplier computes the current multiplier based on elapsed time
func calculateMultiplier(elapsed float64) float64 {
	// Exponential growth formula
	mult := 1.0 + (elapsed / 1.5) + (elapsed * elapsed * 0.005)
	return float64(int(mult*100)) / 100.0
}

// processBet handles a bet request
func (m *Manager) processBet(req BetRequest) {
	resp := BetResponse{}
	defer func() {
		if req.ResponseChan != nil {
			req.ResponseChan <- resp
		}
	}()

	// Validate bet amount
	if req.Amount < MIN_BET_AMOUNT || req.Amount > MAX_BET_AMOUNT {
		resp.Message = fmt.Sprintf("Bet must be between %.2f and %.2f", MIN_BET_AMOUNT, MAX_BET_AMOUNT)
		return
	}

	m.stateMutex.RLock()
	if m.currentRound == nil || m.currentRound.Status != "BETTING" {
		m.stateMutex.RUnlock()
		resp.Message = "Betting is closed"
		return
	}
	roundID := m.currentRound.RoundID
	m.stateMutex.RUnlock()

	// Check user balance (Redis)
	balanceKey := REDIS_KEY_USER_BALANCE + req.UserID
	balance, err := m.redisClient.Get(m.ctx, balanceKey).Float64()
	if err != nil || balance < req.Amount {
		resp.Message = "Insufficient balance"
		resp.Balance = balance
		return
	}

	// Deduct balance atomically (use negative value with IncrByFloat)
	newBalance, err := m.redisClient.IncrByFloat(m.ctx, balanceKey, -req.Amount).Result()
	if err != nil || newBalance < 0 {
		m.redisClient.IncrByFloat(m.ctx, balanceKey, req.Amount) // Rollback
		resp.Message = "Transaction failed"
		return
	}

	// Create bet
	betID := fmt.Sprintf("BET-%s-%d", req.UserID, time.Now().UnixNano())
	bet := ActiveBet{
		BetID:       betID,
		UserID:      req.UserID,
		Amount:      req.Amount,
		AutoCashout: req.AutoCashout,
		PlacedAt:    time.Now(),
		CashedOut:   false,
	}

	// Store in Redis
	betKey := REDIS_KEY_ACTIVE_BETS + roundID
	betJSON, _ := json.Marshal(bet)
	m.redisClient.HSet(m.ctx, betKey, betID, betJSON)
	m.redisClient.Expire(m.ctx, betKey, 10*time.Minute)

	resp.Success = true
	resp.BetID = betID
	resp.Balance = newBalance
	resp.Message = "Bet placed successfully"

	// Broadcast bet placed
	m.hub.Broadcast(map[string]interface{}{
		"type": "bet_placed",
		"data": BetPlacedMessage{
			UserID: req.UserID,
			Amount: req.Amount,
			BetID:  betID,
		},
	})

	log.Printf("[BET] User %s placed %.2f (ID: %s)", req.UserID, req.Amount, betID)
}

// processCashout handles a cashout request
func (m *Manager) processCashout(req CashoutRequest) {
	resp := CashoutResponse{}
	defer func() {
		if req.ResponseChan != nil {
			req.ResponseChan <- resp
		}
	}()

	m.stateMutex.RLock()
	if m.currentRound == nil || m.currentRound.Status != "RUNNING" {
		m.stateMutex.RUnlock()
		resp.Message = "Cannot cashout now"
		return
	}
	currentMult := m.currentRound.CurrentMultiplier
	roundID := m.currentRound.RoundID
	m.stateMutex.RUnlock()

	// Get bet from Redis
	betKey := REDIS_KEY_ACTIVE_BETS + roundID
	betJSON, err := m.redisClient.HGet(m.ctx, betKey, req.BetID).Result()
	if err != nil {
		resp.Message = "Bet not found"
		return
	}

	var bet ActiveBet
	json.Unmarshal([]byte(betJSON), &bet)

	if bet.CashedOut {
		resp.Message = "Already cashed out"
		return
	}

	// Calculate payout
	payout := bet.Amount * currentMult

	// Credit user balance
	balanceKey := REDIS_KEY_USER_BALANCE + req.UserID
	newBalance, err := m.redisClient.IncrByFloat(m.ctx, balanceKey, payout).Result()
	if err != nil {
		resp.Message = "Failed to credit balance"
		return
	}

	// Mark as cashed out
	bet.CashedOut = true
	betJSONBytes, _ := json.Marshal(bet)
	m.redisClient.HSet(m.ctx, betKey, req.BetID, string(betJSONBytes))

	resp.Success = true
	resp.Multiplier = currentMult
	resp.Payout = payout
	resp.Balance = newBalance
	resp.Message = fmt.Sprintf("Cashed out at %.2fx", currentMult)

	// Broadcast cashout
	m.hub.Broadcast(map[string]interface{}{
		"type": "cashout",
		"data": CashoutMessage{
			UserID:     req.UserID,
			BetID:      req.BetID,
			Multiplier: currentMult,
			Payout:     payout,
		},
	})

	log.Printf("[CASHOUT] User %s cashed out at %.2fx (Payout: %.2f)", req.UserID, currentMult, payout)
}

// processAutoCashouts checks and processes auto-cashout targets
func (m *Manager) processAutoCashouts(roundID string, currentMult float64, bets map[string]ActiveBet) {
	for betID, bet := range bets {
		if !bet.CashedOut && bet.AutoCashout > 0 && currentMult >= bet.AutoCashout {
			go m.processCashout(CashoutRequest{
				UserID:  bet.UserID,
				BetID:   betID,
				RoundID: roundID,
			})
		}
	}
}

// processRoundEnd handles end-of-round cleanup
func (m *Manager) processRoundEnd(roundID string, bets map[string]ActiveBet) {
	log.Printf("[ROUND END] Processing %d remaining bets", len(bets))

	for _, bet := range bets {
		if !bet.CashedOut {
			log.Printf("[LOSS] User %s lost %.2f", bet.UserID, bet.Amount)
		}
	}

	// Clear Redis active bets
	betKey := REDIS_KEY_ACTIVE_BETS + roundID
	m.redisClient.Del(m.ctx, betKey)
}

// storeRoundInRedis stores round data in Redis
func (m *Manager) storeRoundInRedis(round *RoundState) {
	key := REDIS_KEY_ROUND_PREFIX + round.RoundID
	data, _ := json.Marshal(round)
	m.redisClient.Set(m.ctx, key, data, 1*time.Hour)
}

// loadActiveBets loads active bets from Redis
func (m *Manager) loadActiveBets(roundID string) map[string]ActiveBet {
	bets := make(map[string]ActiveBet)
	betKey := REDIS_KEY_ACTIVE_BETS + roundID

	result, err := m.redisClient.HGetAll(m.ctx, betKey).Result()
	if err != nil {
		return bets
	}

	for betID, betJSON := range result {
		var bet ActiveBet
		if json.Unmarshal([]byte(betJSON), &bet) == nil {
			bets[betID] = bet
		}
	}

	return bets
}
