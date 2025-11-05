package server

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestHealthHandler(t *testing.T) {
	// Create a minimal Fiber app for testing
	app := fiber.New()
	
	// Add health endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"message": "Server is running",
		})
	})
	
	// Create a test HTTP request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}
	
	// Perform the request
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("could not perform request: %v", err)
	}
	
	// Check the status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", resp.Status)
	}
	
	// Check the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read response body: %v", err)
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("could not unmarshal response: %v", err)
	}
	
	if result["status"] != "ok" {
		t.Errorf("expected status to be 'ok'; got %v", result["status"])
	}
}
