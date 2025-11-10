package game

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

type GameType string

const (
	GameTypeAviator GameType = "aviator"
	GameTypeMines   GameType = "mines"
	GameTypePlinko  GameType = "plinko"
	GameTypeDice    GameType = "dice"
)

type GameEngine interface {
	GetType() GameType
	Start(ctx context.Context) error
	Stop() error
	GetState() interface{}
	PlaceBet(ctx context.Context, req interface{}) (interface{}, error)
	ProcessAction(ctx context.Context, action string, req interface{}) (interface{}, error)
}

type GameFactory struct {
	engines      map[GameType]GameEngine
	redisClient  *redis.Client
	hub          *Hub
	ctx          context.Context
}

func NewGameFactory(redisClient *redis.Client, hub *Hub) *GameFactory {
	return &GameFactory{
		engines:     make(map[GameType]GameEngine),
		redisClient: redisClient,
		hub:         hub,
		ctx:         context.Background(),
	}
}

func (gf *GameFactory) RegisterEngine(engine GameEngine) {
	gf.engines[engine.GetType()] = engine
}

func (gf *GameFactory) GetEngine(gameType GameType) (GameEngine, bool) {
	engine, exists := gf.engines[gameType]
	return engine, exists
}

func (gf *GameFactory) StartAll() error {
	for gameType, engine := range gf.engines {
		if err := engine.Start(gf.ctx); err != nil {
			return err
		}
		log.Printf("[FACTORY] Started %s engine", gameType)
	}
	return nil
}

func (gf *GameFactory) StopAll() error {
	for gameType, engine := range gf.engines {
		if err := engine.Stop(); err != nil {
			return err
		}
		log.Printf("[FACTORY] Stopped %s engine", gameType)
	}
	return nil
}
