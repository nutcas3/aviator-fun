package server

import (
	"aviator/internal/game"

	"github.com/gofiber/fiber/v2"
)

// RegisterGameRoutes registers routes for all game types
func (s *FiberServer) RegisterGameRoutes() {
	api := s.App.Group("/api/v1")

	// Mines game routes
	mines := api.Group("/mines")
	mines.Post("/bet", s.minesBetHandler)
	mines.Post("/click", s.minesClickHandler)
	mines.Post("/cashout", s.minesCashoutHandler)

	// Plinko game routes
	plinko := api.Group("/plinko")
	plinko.Post("/drop", s.plinkoDropHandler)

	// Dice game routes
	dice := api.Group("/dice")
	dice.Post("/roll", s.diceRollHandler)
}

func (s *FiberServer) minesBetHandler(c *fiber.Ctx) error {
	var req game.MinesBetRequest
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

	// Get Mines engine from factory
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

	// Validate required fields
	if req.UserID == "" || req.GameID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "User ID and Game ID are required",
		})
	}

	// Get Mines engine from factory
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

	// Validate required fields
	if req.UserID == "" || req.GameID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "User ID and Game ID are required",
		})
	}

	// Get Mines engine from factory
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

// Plinko Handlers

func (s *FiberServer) plinkoDropHandler(c *fiber.Ctx) error {
	var req game.PlinkoDropRequest
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

	// Get Plinko engine from factory
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

// Dice Handlers

func (s *FiberServer) diceRollHandler(c *fiber.Ctx) error {
	var req game.DiceRollRequest
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

	// Get Dice engine from factory
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
