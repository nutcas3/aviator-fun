package server

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2/middleware/cors"
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

	// Aviator game routes
	api := s.App.Group("/api/v1")

	api.Get("/game/state", s.getGameStateHandler)
	api.Post("/game/bet", s.placeBetHandler)
	api.Post("/game/cashout", s.cashoutHandler)
	api.Get("/user/:userId/balance", s.getUserBalanceHandler)
	api.Post("/user/:userId/balance", s.setUserBalanceHandler)

	// Register new game routes (Mines, Plinko, Dice)
	s.RegisterGameRoutes()

	// WebSocket route
	s.App.Get("/ws", websocket.New(s.gameWebSocketHandler))
}
