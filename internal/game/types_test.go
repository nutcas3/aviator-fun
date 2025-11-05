package game

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBetRequest_JSON(t *testing.T) {
	req := BetRequest{
		UserID:      "user123",
		Amount:      100.50,
		AutoCashout: 2.5,
		RoundID:     "round_001",
	}

	// Marshal to JSON
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal BetRequest: %v", err)
	}

	// Unmarshal back
	var decoded BetRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal BetRequest: %v", err)
	}

	// Verify fields
	if decoded.UserID != req.UserID {
		t.Errorf("UserID = %v, want %v", decoded.UserID, req.UserID)
	}
	if decoded.Amount != req.Amount {
		t.Errorf("Amount = %v, want %v", decoded.Amount, req.Amount)
	}
	if decoded.AutoCashout != req.AutoCashout {
		t.Errorf("AutoCashout = %v, want %v", decoded.AutoCashout, req.AutoCashout)
	}
}

func TestBetResponse_JSON(t *testing.T) {
	resp := BetResponse{
		Success: true,
		Message: "Bet placed successfully",
		BetID:   "bet_123",
		Balance: 9900.50,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal BetResponse: %v", err)
	}

	var decoded BetResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal BetResponse: %v", err)
	}

	if decoded.Success != resp.Success {
		t.Errorf("Success = %v, want %v", decoded.Success, resp.Success)
	}
	if decoded.BetID != resp.BetID {
		t.Errorf("BetID = %v, want %v", decoded.BetID, resp.BetID)
	}
}

func TestCashoutRequest_JSON(t *testing.T) {
	req := CashoutRequest{
		UserID:  "user456",
		BetID:   "bet_789",
		RoundID: "round_002",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal CashoutRequest: %v", err)
	}

	var decoded CashoutRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal CashoutRequest: %v", err)
	}

	if decoded.UserID != req.UserID {
		t.Errorf("UserID = %v, want %v", decoded.UserID, req.UserID)
	}
	if decoded.BetID != req.BetID {
		t.Errorf("BetID = %v, want %v", decoded.BetID, req.BetID)
	}
}

func TestCashoutResponse_JSON(t *testing.T) {
	resp := CashoutResponse{
		Success:    true,
		Message:    "Cashed out successfully",
		Multiplier: 2.45,
		Payout:     245.00,
		Balance:    10145.00,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal CashoutResponse: %v", err)
	}

	var decoded CashoutResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal CashoutResponse: %v", err)
	}

	if decoded.Multiplier != resp.Multiplier {
		t.Errorf("Multiplier = %v, want %v", decoded.Multiplier, resp.Multiplier)
	}
	if decoded.Payout != resp.Payout {
		t.Errorf("Payout = %v, want %v", decoded.Payout, resp.Payout)
	}
}

func TestRoundState_JSON(t *testing.T) {
	now := time.Now()
	state := RoundState{
		RoundID:           "round_123",
		ServerSeed:        "secret_seed",
		HashCommitment:    "abc123def456",
		ClientSeed:        "client_seed_789",
		CrashMultiplier:   3.14,
		CurrentMultiplier: 2.50,
		Status:            "RUNNING",
		StartTime:         now,
		Nonce:             42,
	}

	data, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("Failed to marshal RoundState: %v", err)
	}

	// Verify server seed is not exposed (has json:"-" tag)
	var jsonMap map[string]interface{}
	if err := json.Unmarshal(data, &jsonMap); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, exists := jsonMap["server_seed"]; exists {
		t.Error("ServerSeed should not be in JSON output")
	}

	if _, exists := jsonMap["crash_multiplier"]; exists {
		t.Error("CrashMultiplier should not be in JSON output")
	}

	// Verify other fields are present
	if jsonMap["round_id"] != state.RoundID {
		t.Errorf("round_id = %v, want %v", jsonMap["round_id"], state.RoundID)
	}
	if jsonMap["status"] != state.Status {
		t.Errorf("status = %v, want %v", jsonMap["status"], state.Status)
	}
}

func TestActiveBet_JSON(t *testing.T) {
	now := time.Now()
	bet := ActiveBet{
		BetID:       "bet_001",
		UserID:      "user_001",
		Amount:      100.00,
		AutoCashout: 2.0,
		PlacedAt:    now,
		CashedOut:   false,
	}

	data, err := json.Marshal(bet)
	if err != nil {
		t.Fatalf("Failed to marshal ActiveBet: %v", err)
	}

	var decoded ActiveBet
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ActiveBet: %v", err)
	}

	if decoded.BetID != bet.BetID {
		t.Errorf("BetID = %v, want %v", decoded.BetID, bet.BetID)
	}
	if decoded.Amount != bet.Amount {
		t.Errorf("Amount = %v, want %v", decoded.Amount, bet.Amount)
	}
	if decoded.CashedOut != bet.CashedOut {
		t.Errorf("CashedOut = %v, want %v", decoded.CashedOut, bet.CashedOut)
	}
}

func TestWSMessage_JSON(t *testing.T) {
	msg := WSMessage{
		Type: "test_message",
		Data: map[string]interface{}{
			"key": "value",
			"num": 123,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal WSMessage: %v", err)
	}

	var decoded WSMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal WSMessage: %v", err)
	}

	if decoded.Type != msg.Type {
		t.Errorf("Type = %v, want %v", decoded.Type, msg.Type)
	}
}

func TestBetPlacedMessage_JSON(t *testing.T) {
	msg := BetPlacedMessage{
		UserID: "user_123",
		Amount: 500.00,
		BetID:  "bet_456",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal BetPlacedMessage: %v", err)
	}

	var decoded BetPlacedMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal BetPlacedMessage: %v", err)
	}

	if decoded.UserID != msg.UserID {
		t.Errorf("UserID = %v, want %v", decoded.UserID, msg.UserID)
	}
	if decoded.Amount != msg.Amount {
		t.Errorf("Amount = %v, want %v", decoded.Amount, msg.Amount)
	}
}

func TestCashoutMessage_JSON(t *testing.T) {
	msg := CashoutMessage{
		UserID:     "user_789",
		BetID:      "bet_101",
		Multiplier: 3.50,
		Payout:     350.00,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal CashoutMessage: %v", err)
	}

	var decoded CashoutMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal CashoutMessage: %v", err)
	}

	if decoded.Multiplier != msg.Multiplier {
		t.Errorf("Multiplier = %v, want %v", decoded.Multiplier, msg.Multiplier)
	}
	if decoded.Payout != msg.Payout {
		t.Errorf("Payout = %v, want %v", decoded.Payout, msg.Payout)
	}
}
