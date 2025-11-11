package server


func (s *FiberServer) RegisterGameRoutes() {
	api := s.App.Group("/api/v1")

	// Aviator game routes
	api.Get("/game/state", s.getGameStateHandler)
	api.Post("/game/bet", s.placeBetHandler)
	api.Post("/game/cashout", s.cashoutHandler)

	// User balance routes
	api.Get("/user/:userId/balance", s.getUserBalanceHandler)
	api.Post("/user/:userId/balance", s.setUserBalanceHandler)

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
