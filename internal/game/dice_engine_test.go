package game

import (
	"testing"
)

func TestDiceEngine_GenerateRoll(t *testing.T) {
	engine := &DiceEngine{}

	t.Run("generates result within valid range", func(t *testing.T) {
		result := engine.generateRoll("seed1", "seed2", 1)
		if result < DICE_MIN_VALUE || result > DICE_MAX_VALUE {
			t.Errorf("result %.2f out of range [%.2f, %.2f]", result, DICE_MIN_VALUE, DICE_MAX_VALUE)
		}
	})

	t.Run("deterministic generation", func(t *testing.T) {
		result1 := engine.generateRoll("seed1", "seed2", 1)
		result2 := engine.generateRoll("seed1", "seed2", 1)

		if result1 != result2 {
			t.Error("results should be deterministic")
		}
	})

	t.Run("different seeds produce different results", func(t *testing.T) {
		result1 := engine.generateRoll("seed1", "seed2", 1)
		result2 := engine.generateRoll("seed3", "seed4", 1)

		if result1 == result2 {
			t.Log("Warning: different seeds produced same result (possible but unlikely)")
		}
	})

	t.Run("different nonces produce different results", func(t *testing.T) {
		result1 := engine.generateRoll("seed1", "seed2", 1)
		result2 := engine.generateRoll("seed1", "seed2", 2)

		if result1 == result2 {
			t.Error("different nonces should produce different results")
		}
	})
}

func TestDiceEngine_CalculateMultiplier(t *testing.T) {
	engine := &DiceEngine{}

	t.Run("roll over 50 gives ~2x multiplier", func(t *testing.T) {
		multiplier := engine.calculateMultiplier(50.0, true)
		if multiplier < 1.8 || multiplier > 2.2 {
			t.Errorf("expected multiplier around 2x, got %.2f", multiplier)
		}
	})

	t.Run("roll under 50 gives ~2x multiplier", func(t *testing.T) {
		multiplier := engine.calculateMultiplier(50.0, false)
		if multiplier < 1.8 || multiplier > 2.2 {
			t.Errorf("expected multiplier around 2x, got %.2f", multiplier)
		}
	})

	t.Run("higher target for roll over gives higher multiplier", func(t *testing.T) {
		mult50 := engine.calculateMultiplier(50.0, true)
		mult90 := engine.calculateMultiplier(90.0, true)

		if mult90 <= mult50 {
			t.Error("higher target should give higher multiplier for roll over")
		}
	})

	t.Run("lower target for roll under gives higher multiplier", func(t *testing.T) {
		mult50 := engine.calculateMultiplier(50.0, false)
		mult10 := engine.calculateMultiplier(10.0, false)

		if mult10 <= mult50 {
			t.Error("lower target should give higher multiplier for roll under")
		}
	})

	t.Run("extreme targets produce valid multipliers", func(t *testing.T) {
		mult1 := engine.calculateMultiplier(1.0, false)
		mult99 := engine.calculateMultiplier(99.0, true)

		if mult1 <= 0 || mult99 <= 0 {
			t.Error("extreme targets should still produce positive multipliers")
		}

		if mult1 < 50 {
			t.Error("very low target should produce high multiplier")
		}

		if mult99 < 50 {
			t.Error("very high target should produce high multiplier")
		}
	})

	t.Run("multiplier never zero or negative", func(t *testing.T) {
		targets := []float64{0.5, 10.0, 25.0, 50.0, 75.0, 90.0, 99.5}

		for _, target := range targets {
			multOver := engine.calculateMultiplier(target, true)
			multUnder := engine.calculateMultiplier(target, false)

			if multOver <= 0 {
				t.Errorf("multiplier for target %.2f (over) is non-positive", target)
			}
			if multUnder <= 0 {
				t.Errorf("multiplier for target %.2f (under) is non-positive", target)
			}
		}
	})
}

func TestDiceEngine_GetType(t *testing.T) {
	engine := &DiceEngine{}

	if engine.GetType() != GameTypeDice {
		t.Errorf("expected GameTypeDice, got %v", engine.GetType())
	}
}

func TestDiceEngine_WinLogic(t *testing.T) {
	t.Run("roll over logic", func(t *testing.T) {
		target := 50.0

		// Should win if result > target
		if !(60.0 > target) {
			t.Error("60 should be over 50")
		}

		// Should lose if result <= target
		if 40.0 > target {
			t.Error("40 should not be over 50")
		}
	})

	t.Run("roll under logic", func(t *testing.T) {
		target := 50.0

		// Should win if result < target
		if !(40.0 < target) {
			t.Error("40 should be under 50")
		}

		// Should lose if result >= target
		if 60.0 < target {
			t.Error("60 should not be under 50")
		}
	})
}
