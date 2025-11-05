package server

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/gofiber/contrib/websocket"

	"aviator/internal/game"
)

func (s *FiberServer) RegisterFiberRoutes() {
	// Apply CORS middleware
	s.App.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Accept,Authorization,Content-Type",
		AllowCredentials: false, // credentials require explicit origins
		MaxAge:           300,
	}))

	// Basic routes
	s.App.Get("/health", s.healthHandler)

	// Game routes
	api := s.App.Group("/api/v1")
	
	api.Get("/game/state", s.getGameStateHandler)
	api.Post("/game/bet", s.placeBetHandler)
	api.Post("/game/cashout", s.cashoutHandler)
	api.Get("/user/:userId/balance", s.getUserBalanceHandler)
	api.Post("/user/:userId/balance", s.setUserBalanceHandler)

	// WebSocket route
	s.App.Get("/ws", websocket.New(s.gameWebSocketHandler))

}


func (s *FiberServer) healthHandler(c *fiber.Ctx) error {
	health := fiber.Map{
		"database": s.db.Health(),
		"cache":    s.cache.Health(),
		"game": fiber.Map{
			"status":           "running",
			"connected_clients": s.gameHub.GetClientCount(),
		},
	}
	return c.JSON(health)
}

// getGameStateHandler returns the current game state
func (s *FiberServer) getGameStateHandler(c *fiber.Ctx) error {
	state := s.gameManager.GetCurrentRound()
	if state == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "No active game round",
		})
	}
	return c.JSON(state)
}

// placeBetHandler handles bet placement requests
func (s *FiberServer) placeBetHandler(c *fiber.Ctx) error {
	var req game.BetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate user ID
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

// cashoutHandler handles cashout requests
func (s *FiberServer) cashoutHandler(c *fiber.Ctx) error {
	var req game.CashoutRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
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

// getUserBalanceHandler returns a user's balance
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

// setUserBalanceHandler sets a user's balance (for testing/admin)
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

// gameWebSocketHandler handles WebSocket connections for real-time game updates
func (s *FiberServer) gameWebSocketHandler(conn *websocket.Conn) {
	// Extract user ID from query params
	userID := conn.Query("user_id", "anonymous")

	log.Printf("[WS] New connection from user: %s", userID)

	// Register client with hub
	s.gameHub.RegisterClient(conn, userID)

	// Send initial state
	currentState := s.gameManager.GetCurrentRound()
	if currentState != nil {
		stateJSON, _ := json.Marshal(map[string]interface{}{
			"type": "initial_state",
			"data": currentState,
		})
		conn.WriteMessage(websocket.TextMessage, stateJSON)
	}

	// Handle incoming messages
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
