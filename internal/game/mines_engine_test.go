package game

import (
	"testing"
)

func TestMinesEngine_GenerateMinePositions(t *testing.T) {
	engine := &MinesEngine{}

	t.Run("generates correct number of mines", func(t *testing.T) {
		positions := engine.generateMinePositions("seed1", "seed2", 1, 5)
		if len(positions) != 5 {
			t.Errorf("expected 5 positions, got %d", len(positions))
		}
	})

	t.Run("generates unique positions", func(t *testing.T) {
		positions := engine.generateMinePositions("seed1", "seed2", 1, 10)
		uniqueMap := make(map[int]bool)
		for _, pos := range positions {
			uniqueMap[pos] = true
		}
		if len(uniqueMap) != 10 {
			t.Errorf("expected 10 unique positions, got %d", len(uniqueMap))
		}
	})

	t.Run("positions within grid bounds", func(t *testing.T) {
		positions := engine.generateMinePositions("seed1", "seed2", 1, 15)
		for _, pos := range positions {
			if pos < 0 || pos >= MINES_GRID_SIZE {
				t.Errorf("position %d out of bounds [0, %d)", pos, MINES_GRID_SIZE)
			}
		}
	})

	t.Run("deterministic generation", func(t *testing.T) {
		positions1 := engine.generateMinePositions("seed1", "seed2", 1, 5)
		positions2 := engine.generateMinePositions("seed1", "seed2", 1, 5)
		
		if len(positions1) != len(positions2) {
			t.Error("positions should be deterministic")
		}
		
		for i := range positions1 {
			if positions1[i] != positions2[i] {
				t.Error("positions should be deterministic")
				break
			}
		}
	})
}

func TestMinesEngine_CalculatePayout(t *testing.T) {
	engine := &MinesEngine{}

	t.Run("payout increases with revealed tiles", func(t *testing.T) {
		payout0 := engine.calculatePayout(100.0, 3, 0)
		payout1 := engine.calculatePayout(100.0, 3, 1)
		payout2 := engine.calculatePayout(100.0, 3, 2)

		if payout0 != 100.0 {
			t.Errorf("expected initial payout 100.0, got %.2f", payout0)
		}
		if payout1 <= payout0 {
			t.Error("payout should increase with revealed tiles")
		}
		if payout2 <= payout1 {
			t.Error("payout should continue increasing")
		}
	})

	t.Run("higher mine count increases multiplier", func(t *testing.T) {
		payout3Mines := engine.calculatePayout(100.0, 3, 5)
		payout10Mines := engine.calculatePayout(100.0, 10, 5)

		if payout10Mines <= payout3Mines {
			t.Error("higher mine count should result in higher payout")
		}
	})

	t.Run("zero revealed tiles returns bet amount", func(t *testing.T) {
		payout := engine.calculatePayout(250.0, 5, 0)
		if payout != 250.0 {
			t.Errorf("expected 250.0, got %.2f", payout)
		}
	})
}

func TestMinesEngine_GetType(t *testing.T) {
	engine := &MinesEngine{}
	
	if engine.GetType() != GameTypeMines {
		t.Errorf("expected GameTypeMines, got %v", engine.GetType())
	}
}
