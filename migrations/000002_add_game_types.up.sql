ALTER TABLE game_rounds ADD COLUMN IF NOT EXISTS game_type VARCHAR(20) NOT NULL DEFAULT 'aviator';
ALTER TABLE bets ADD COLUMN IF NOT EXISTS game_type VARCHAR(20) NOT NULL DEFAULT 'aviator';

CREATE TABLE IF NOT EXISTS mines_games (
    id VARCHAR(100) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bet_amount DECIMAL(20,2) NOT NULL,
    mine_count INTEGER NOT NULL,
    server_seed VARCHAR(128) NOT NULL,
    client_seed VARCHAR(128) NOT NULL,
    nonce INTEGER NOT NULL,
    mine_positions INTEGER[] NOT NULL,
    revealed_tiles INTEGER[] NOT NULL DEFAULT '{}',
    current_payout DECIMAL(20,2) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP,
    CONSTRAINT valid_mines_status CHECK (status IN ('ACTIVE', 'CASHED_OUT', 'BUSTED')),
    CONSTRAINT valid_mine_count CHECK (mine_count >= 1 AND mine_count <= 24),
    CONSTRAINT positive_bet_amount CHECK (bet_amount > 0)
);

CREATE TABLE IF NOT EXISTS plinko_games (
    id VARCHAR(100) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bet_amount DECIMAL(20,2) NOT NULL,
    risk VARCHAR(10) NOT NULL,
    rows INTEGER NOT NULL,
    server_seed VARCHAR(128) NOT NULL,
    client_seed VARCHAR(128) NOT NULL,
    nonce INTEGER NOT NULL,
    path INTEGER[] NOT NULL,
    landing_slot INTEGER NOT NULL,
    multiplier DECIMAL(10,2) NOT NULL,
    payout DECIMAL(20,2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_plinko_risk CHECK (risk IN ('low', 'medium', 'high')),
    CONSTRAINT valid_plinko_rows CHECK (rows IN (8, 12, 16)),
    CONSTRAINT positive_plinko_bet CHECK (bet_amount > 0)
);

CREATE TABLE IF NOT EXISTS dice_games (
    id VARCHAR(100) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bet_amount DECIMAL(20,2) NOT NULL,
    target DECIMAL(5,2) NOT NULL,
    is_over BOOLEAN NOT NULL,
    server_seed VARCHAR(128) NOT NULL,
    client_seed VARCHAR(128) NOT NULL,
    nonce INTEGER NOT NULL,
    roll_result DECIMAL(5,2) NOT NULL,
    win BOOLEAN NOT NULL,
    multiplier DECIMAL(10,2) NOT NULL,
    payout DECIMAL(20,2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_dice_target CHECK (target >= 0.00 AND target <= 100.00),
    CONSTRAINT valid_dice_roll CHECK (roll_result >= 0.00 AND roll_result <= 100.00),
    CONSTRAINT positive_dice_bet CHECK (bet_amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_mines_games_user_id ON mines_games(user_id);
CREATE INDEX IF NOT EXISTS idx_mines_games_created_at ON mines_games(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mines_games_status ON mines_games(status);

CREATE INDEX IF NOT EXISTS idx_plinko_games_user_id ON plinko_games(user_id);
CREATE INDEX IF NOT EXISTS idx_plinko_games_created_at ON plinko_games(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_dice_games_user_id ON dice_games(user_id);
CREATE INDEX IF NOT EXISTS idx_dice_games_created_at ON dice_games(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dice_games_win ON dice_games(win);

CREATE INDEX IF NOT EXISTS idx_game_rounds_game_type ON game_rounds(game_type);
CREATE INDEX IF NOT EXISTS idx_bets_game_type ON bets(game_type);

COMMENT ON TABLE mines_games IS 'Stores Mines game sessions with grid state and results';
COMMENT ON TABLE plinko_games IS 'Stores Plinko game results with ball path and landing slot';
COMMENT ON TABLE dice_games IS 'Stores Dice game results with roll outcomes';
COMMENT ON COLUMN game_rounds.game_type IS 'Type of game: aviator, mines, plinko, dice';
COMMENT ON COLUMN bets.game_type IS 'Type of game the bet was placed on';
