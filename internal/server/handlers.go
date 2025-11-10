package server

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	"aviator/internal/game"
)

// Health handler
func (s *FiberServer) healthHandler(c *fiber.Ctx) error {
	health := fiber.Map{
		"database": s.db.Health(),
		"cache":    s.cache.Health(),
		"game": fiber.Map{
			"status":            "running",
			"connected_clients": s.gameHub.GetClientCount(),
		},
	}
	return c.JSON(health)
}

// Aviator game handlers

func (s *FiberServer) getGameStateHandler(c *fiber.Ctx) error {
	state := s.gameManager.GetCurrentRound()
	if state == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "No active game round",
		})
	}
	return c.JSON(state)
}

func (s *FiberServer) placeBetHandler(c *fiber.Ctx) error {
	var req game.BetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.UserID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	resp := s.gameManager.PlaceBet(req)
	if !resp.Success {
		return c.Status(400).JSON(resp)
	}

	return c.JSON(resp)
}

func (s *FiberServer) cashoutHandler(c *fiber.Ctx) error {
	var req game.CashoutRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.UserID == "" || req.BetID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "User ID and Bet ID are required",
		})
	}

	resp := s.gameManager.Cashout(req)
	if !resp.Success {
		return c.Status(400).JSON(resp)
	}

	return c.JSON(resp)
}

// User balance handlers

func (s *FiberServer) getUserBalanceHandler(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	balanceKey := game.REDIS_KEY_USER_BALANCE + userID
	balance, err := s.cache.GetClient().Get(c.Context(), balanceKey).Float64()
	if err != nil {
		balance = 0.0
	}

	return c.JSON(fiber.Map{
		"user_id": userID,
		"balance": balance,
	})
}

func (s *FiberServer) setUserBalanceHandler(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	var body struct {
		Balance float64 `json:"balance"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	balanceKey := game.REDIS_KEY_USER_BALANCE + userID
	err := s.cache.GetClient().Set(c.Context(), balanceKey, body.Balance, 0).Err()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to set balance",
		})
	}

	return c.JSON(fiber.Map{
		"user_id": userID,
		"balance": body.Balance,
		"message": "Balance updated successfully",
	})
}

// Mines game handlers

func (s *FiberServer) minesBetHandler(c *fiber.Ctx) error {
	var req game.MinesBetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.UserID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	engine, exists := s.gameFactory.GetEngine(game.GameTypeMines)
	if !exists {
		return c.Status(500).JSON(fiber.Map{
			"error": "Mines game not available",
		})
	}

	resp, err := engine.PlaceBet(c.Context(), req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	betResp, ok := resp.(game.MinesBetResponse)
	if !ok || !betResp.Success {
		return c.Status(400).JSON(resp)
	}

	return c.JSON(resp)
}

func (s *FiberServer) minesClickHandler(c *fiber.Ctx) error {
	var req game.MinesClickRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.UserID == "" || req.GameID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "User ID and Game ID are required",
		})
	}

	engine, exists := s.gameFactory.GetEngine(game.GameTypeMines)
	if !exists {
		return c.Status(500).JSON(fiber.Map{
			"error": "Mines game not available",
		})
	}

	resp, err := engine.ProcessAction(c.Context(), "click", req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	clickResp, ok := resp.(game.MinesClickResponse)
	if !ok || !clickResp.Success {
		return c.Status(400).JSON(resp)
	}

	return c.JSON(resp)
}

func (s *FiberServer) minesCashoutHandler(c *fiber.Ctx) error {
	var req game.MinesCashoutRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.UserID == "" || req.GameID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "User ID and Game ID are required",
		})
	}

	engine, exists := s.gameFactory.GetEngine(game.GameTypeMines)
	if !exists {
		return c.Status(500).JSON(fiber.Map{
			"error": "Mines game not available",
		})
	}

	resp, err := engine.ProcessAction(c.Context(), "cashout", req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	cashoutResp, ok := resp.(game.MinesCashoutResponse)
	if !ok || !cashoutResp.Success {
		return c.Status(400).JSON(resp)
	}

	return c.JSON(resp)
}

// Plinko game handlers

func (s *FiberServer) plinkoDropHandler(c *fiber.Ctx) error {
	var req game.PlinkoDropRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.UserID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	engine, exists := s.gameFactory.GetEngine(game.GameTypePlinko)
	if !exists {
		return c.Status(500).JSON(fiber.Map{
			"error": "Plinko game not available",
		})
	}

	resp, err := engine.PlaceBet(c.Context(), req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	dropResp, ok := resp.(game.PlinkoDropResponse)
	if !ok || !dropResp.Success {
		return c.Status(400).JSON(resp)
	}

	return c.JSON(resp)
}

// Dice game handlers

func (s *FiberServer) diceRollHandler(c *fiber.Ctx) error {
	var req game.DiceRollRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.UserID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	engine, exists := s.gameFactory.GetEngine(game.GameTypeDice)
	if !exists {
		return c.Status(500).JSON(fiber.Map{
			"error": "Dice game not available",
		})
	}

	resp, err := engine.PlaceBet(c.Context(), req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	rollResp, ok := resp.(game.DiceRollResponse)
	if !ok || !rollResp.Success {
		return c.Status(400).JSON(resp)
	}

	return c.JSON(resp)
}

// WebSocket handler

func (s *FiberServer) gameWebSocketHandler(conn *websocket.Conn) {
	userID := conn.Query("user_id", "anonymous")

	log.Printf("[WS] New connection from user: %s", userID)

	s.gameHub.RegisterClient(conn, userID)

	currentState := s.gameManager.GetCurrentRound()
	if currentState != nil {
		stateJSON, _ := json.Marshal(map[string]interface{}{
			"type": "initial_state",
			"data": currentState,
		})
		conn.WriteMessage(websocket.TextMessage, stateJSON)
	}

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[WS] Read error for user %s: %v", userID, err)
			s.gameHub.UnregisterClient(conn)
			break
		}

		if messageType == websocket.TextMessage {
			var clientMsg map[string]interface{}
			if err := json.Unmarshal(message, &clientMsg); err != nil {
				continue
			}

			msgType, ok := clientMsg["type"].(string)
			if !ok {
				continue
			}

			switch msgType {
			case "place_bet":
				amount, _ := strconv.ParseFloat(fmt.Sprintf("%v", clientMsg["amount"]), 64)
				autoCashout, _ := strconv.ParseFloat(fmt.Sprintf("%v", clientMsg["auto_cashout"]), 64)

				resp := s.gameManager.PlaceBet(game.BetRequest{
					UserID:      userID,
					Amount:      amount,
					AutoCashout: autoCashout,
				})

				respJSON, _ := json.Marshal(resp)
				conn.WriteMessage(websocket.TextMessage, respJSON)

			case "cashout":
				betID := fmt.Sprintf("%v", clientMsg["bet_id"])

				resp := s.gameManager.Cashout(game.CashoutRequest{
					UserID: userID,
					BetID:  betID,
				})

				respJSON, _ := json.Marshal(resp)
				conn.WriteMessage(websocket.TextMessage, respJSON)

			case "ping":
				pongJSON, _ := json.Marshal(map[string]string{"type": "pong"})
				conn.WriteMessage(websocket.TextMessage, pongJSON)
			}
		}
	}
}
