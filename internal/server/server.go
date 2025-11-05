package server

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"aviator/internal/cache"
	"aviator/internal/database"
	"aviator/internal/game"
)

type FiberServer struct {
	*fiber.App

	db          database.Service
	cache       cache.Service
	gameManager *game.Manager
	gameHub     *game.Hub
}

func New() *FiberServer {
	// Initialize database
	db := database.New()

	// Initialize Redis cache
	redisService := cache.New()
	if redisService == nil {
		log.Fatal("[SERVER] Redis is required for game functionality")
	}

	// Initialize game components
	hub := game.NewHub()
	manager := game.NewManager(hub, redisService.GetClient())

	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader:  "aviator",
			AppName:       "aviator",
			ReadTimeout:   10 * time.Second,
			WriteTimeout:  10 * time.Second,
			IdleTimeout:   120 * time.Second,
			StrictRouting: false,
		}),

		db:          db,
		cache:       redisService,
		gameManager: manager,
		gameHub:     hub,
	}

	// Apply global middleware
	server.App.Use(recover.New())
	server.App.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
	}))

	// Start game components
	go hub.Run()
	go manager.Start()

	log.Println("[SERVER] Game manager started")

	return server
}

// Shutdown gracefully shuts down the server and game components
func (s *FiberServer) Shutdown() error {
	log.Println("[SERVER] Shutting down...")

	// Stop game manager
	if s.gameManager != nil {
		s.gameManager.Stop()
	}

	// Close connections
	if s.cache != nil {
		s.cache.Close()
	}
	if s.db != nil {
		s.db.Close()
	}

	return nil
}
