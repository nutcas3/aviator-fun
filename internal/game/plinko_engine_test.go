package game

import (
	"testing"
)

func TestPlinkoEngine_GeneratePath(t *testing.T) {
	engine := &PlinkoEngine{}

	t.Run("generates correct path length", func(t *testing.T) {
		path, _ := engine.generatePath("seed1", "seed2", 1, 16)
		if len(path) != 16 {
			t.Errorf("expected path length 16, got %d", len(path))
		}
	})

	t.Run("path contains only 0 and 1", func(t *testing.T) {
		path, _ := engine.generatePath("seed1", "seed2", 1, 16)
		for i, direction := range path {
			if direction != 0 && direction != 1 {
				t.Errorf("invalid direction %d at position %d", direction, i)
			}
		}
	})

	t.Run("landing slot within bounds", func(t *testing.T) {
		_, landingSlot := engine.generatePath("seed1", "seed2", 1, 16)
		if landingSlot < 0 || landingSlot > 16 {
			t.Errorf("landing slot %d out of bounds", landingSlot)
		}
	})

	t.Run("deterministic generation", func(t *testing.T) {
		path1, slot1 := engine.generatePath("seed1", "seed2", 1, 16)
		path2, slot2 := engine.generatePath("seed1", "seed2", 1, 16)

		if len(path1) != len(path2) {
			t.Error("paths should be deterministic")
		}

		for i := range path1 {
			if path1[i] != path2[i] {
				t.Error("paths should be deterministic")
				break
			}
		}

		if slot1 != slot2 {
			t.Error("landing slots should be deterministic")
		}
	})

	t.Run("different seeds produce different results", func(t *testing.T) {
		_, slot1 := engine.generatePath("seed1", "seed2", 1, 16)
		_, slot2 := engine.generatePath("seed3", "seed4", 1, 16)

		// While not guaranteed, different seeds should usually produce different results
		// This is a probabilistic test
		if slot1 == slot2 {
			t.Log("Warning: different seeds produced same result (possible but unlikely)")
		}
	})
}

func TestPlinkoEngine_GetMultiplier(t *testing.T) {
	engine := &PlinkoEngine{}

	t.Run("returns valid multiplier for low risk", func(t *testing.T) {
		multiplier := engine.getMultiplier(PlinkoRiskLow, 8, 16)
		if multiplier <= 0 {
			t.Error("multiplier should be positive")
		}
	})

	t.Run("returns valid multiplier for medium risk", func(t *testing.T) {
		multiplier := engine.getMultiplier(PlinkoRiskMedium, 8, 16)
		if multiplier <= 0 {
			t.Error("multiplier should be positive")
		}
	})

	t.Run("returns valid multiplier for high risk", func(t *testing.T) {
		multiplier := engine.getMultiplier(PlinkoRiskHigh, 8, 16)
		if multiplier <= 0 {
			t.Error("multiplier should be positive")
		}
	})

	t.Run("handles out of bounds landing slot", func(t *testing.T) {
		multiplier := engine.getMultiplier(PlinkoRiskLow, 999, 16)
		if multiplier <= 0 {
			t.Error("should handle out of bounds gracefully")
		}
	})

	t.Run("handles negative landing slot", func(t *testing.T) {
		multiplier := engine.getMultiplier(PlinkoRiskLow, -1, 16)
		if multiplier <= 0 {
			t.Error("should handle negative values gracefully")
		}
	})
}

func TestPlinkoEngine_GetType(t *testing.T) {
	engine := &PlinkoEngine{}

	if engine.GetType() != GameTypePlinko {
		t.Errorf("expected GameTypePlinko, got %v", engine.GetType())
	}
}

func TestPlinkoEngine_RiskLevels(t *testing.T) {
	t.Run("multipliers exist for all risk levels", func(t *testing.T) {
		risks := []PlinkoRisk{PlinkoRiskLow, PlinkoRiskMedium, PlinkoRiskHigh}

		for _, risk := range risks {
			multipliers, exists := plinkoMultipliers[risk]
			if !exists {
				t.Errorf("multipliers not found for risk level %v", risk)
			}
			if len(multipliers) == 0 {
				t.Errorf("empty multipliers for risk level %v", risk)
			}
		}
	})

	t.Run("high risk has higher max multipliers", func(t *testing.T) {
		lowMax := 0.0
		for _, m := range plinkoMultipliers[PlinkoRiskLow] {
			if m > lowMax {
				lowMax = m
			}
		}

		highMax := 0.0
		for _, m := range plinkoMultipliers[PlinkoRiskHigh] {
			if m > highMax {
				highMax = m
			}
		}

		if highMax <= lowMax {
			t.Error("high risk should have higher maximum multipliers")
		}
	})
}
