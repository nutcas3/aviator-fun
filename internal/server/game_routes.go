package server

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
