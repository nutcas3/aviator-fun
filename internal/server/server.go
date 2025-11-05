package server

import (
	"github.com/gofiber/fiber/v2"

	"aviator/internal/database"
)

type FiberServer struct {
	*fiber.App

	db database.Service
}

func New() *FiberServer {
	server := &FiberServer{
		App: fiber.New(fiber.Config{
			ServerHeader: "aviator",
			AppName:      "aviator",
		}),

		db: database.New(),
	}

	return server
}
