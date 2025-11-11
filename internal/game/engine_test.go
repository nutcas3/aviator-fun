package game

import (
	"testing"

	"github.com/redis/go-redis/v9"
)

func TestGameFactory_RegisterEngine(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})
	hub := NewHub()
	factory := NewGameFactory(client, hub)

	t.Run("register mines engine", func(t *testing.T) {
		minesEngine := NewMinesEngine(client, hub)
		factory.RegisterEngine(minesEngine)

		engine, exists := factory.GetEngine(GameTypeMines)
		if !exists {
			t.Error("mines engine should be registered")
		}
		if engine.GetType() != GameTypeMines {
			t.Error("retrieved engine should be mines type")
		}
	})

	t.Run("register plinko engine", func(t *testing.T) {
		plinkoEngine := NewPlinkoEngine(client, hub)
		factory.RegisterEngine(plinkoEngine)

		engine, exists := factory.GetEngine(GameTypePlinko)
		if !exists {
			t.Error("plinko engine should be registered")
		}
		if engine.GetType() != GameTypePlinko {
			t.Error("retrieved engine should be plinko type")
		}
	})

	t.Run("register dice engine", func(t *testing.T) {
		diceEngine := NewDiceEngine(client, hub)
		factory.RegisterEngine(diceEngine)

		engine, exists := factory.GetEngine(GameTypeDice)
		if !exists {
			t.Error("dice engine should be registered")
		}
		if engine.GetType() != GameTypeDice {
			t.Error("retrieved engine should be dice type")
		}
	})

	t.Run("get non-existent engine", func(t *testing.T) {
		_, exists := factory.GetEngine(GameTypeAviator)
		if exists {
			t.Error("aviator engine should not exist")
		}
	})
}

func TestGameFactory_MultipleEngines(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})
	hub := NewHub()
	factory := NewGameFactory(client, hub)

	// Register all engines
	factory.RegisterEngine(NewMinesEngine(client, hub))
	factory.RegisterEngine(NewPlinkoEngine(client, hub))
	factory.RegisterEngine(NewDiceEngine(client, hub))

	t.Run("all engines accessible", func(t *testing.T) {
		engines := []GameType{GameTypeMines, GameTypePlinko, GameTypeDice}

		for _, gameType := range engines {
			engine, exists := factory.GetEngine(gameType)
			if !exists {
				t.Errorf("engine %v should be registered", gameType)
			}
			if engine.GetType() != gameType {
				t.Errorf("engine type mismatch for %v", gameType)
			}
		}
	})
}

func TestGameType_Constants(t *testing.T) {
	t.Run("game types are unique", func(t *testing.T) {
		types := []GameType{
			GameTypeAviator,
			GameTypeMines,
			GameTypePlinko,
			GameTypeDice,
		}

		uniqueMap := make(map[GameType]bool)
		for _, gameType := range types {
			if uniqueMap[gameType] {
				t.Errorf("duplicate game type: %v", gameType)
			}
			uniqueMap[gameType] = true
		}

		if len(uniqueMap) != len(types) {
			t.Error("game types should all be unique")
		}
	})

	t.Run("game types are non-empty", func(t *testing.T) {
		types := []GameType{
			GameTypeAviator,
			GameTypeMines,
			GameTypePlinko,
			GameTypeDice,
		}

		for _, gameType := range types {
			if string(gameType) == "" {
				t.Errorf("game type should not be empty")
			}
		}
	})
}
