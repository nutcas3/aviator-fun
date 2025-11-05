CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE,
    balance DECIMAL(20,2) NOT NULL DEFAULT 0.00,
    total_wagered DECIMAL(20,2) NOT NULL DEFAULT 0.00,
    total_won DECIMAL(20,2) NOT NULL DEFAULT 0.00,
    total_lost DECIMAL(20,2) NOT NULL DEFAULT 0.00,
    games_played INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    CONSTRAINT positive_balance CHECK (balance >= 0)
);

CREATE TABLE IF NOT EXISTS game_rounds (
    id VARCHAR(50) PRIMARY KEY,
    server_seed VARCHAR(128) NOT NULL,
    hash_commitment VARCHAR(64) NOT NULL,
    client_seed VARCHAR(128) NOT NULL,
    crash_multiplier DECIMAL(10,2) NOT NULL,
    nonce INTEGER NOT NULL,
    started_at TIMESTAMP NOT NULL,
    crashed_at TIMESTAMP,
    status VARCHAR(20) NOT NULL,
    total_bets INTEGER NOT NULL DEFAULT 0,
    total_wagered DECIMAL(20,2) NOT NULL DEFAULT 0.00,
    total_payout DECIMAL(20,2) NOT NULL DEFAULT 0.00,
    CONSTRAINT valid_status CHECK (status IN ('BETTING', 'RUNNING', 'CRASHED')),
    CONSTRAINT valid_multiplier CHECK (crash_multiplier >= 1.00)
);

CREATE TABLE IF NOT EXISTS bets (
    id VARCHAR(100) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    round_id VARCHAR(50) NOT NULL REFERENCES game_rounds(id) ON DELETE CASCADE,
    amount DECIMAL(20,2) NOT NULL,
    auto_cashout DECIMAL(10,2),
    cashout_multiplier DECIMAL(10,2),
    payout DECIMAL(20,2),
    placed_at TIMESTAMP NOT NULL,
    cashed_out_at TIMESTAMP,
    result VARCHAR(20) NOT NULL,
    profit DECIMAL(20,2),
    CONSTRAINT valid_result CHECK (result IN ('WIN', 'LOSS', 'PENDING')),
    CONSTRAINT positive_amount CHECK (amount > 0),
    CONSTRAINT valid_auto_cashout CHECK (auto_cashout IS NULL OR auto_cashout >= 1.01)
);

CREATE TABLE IF NOT EXISTS user_statistics (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    total_bets INTEGER NOT NULL DEFAULT 0,
    total_wins INTEGER NOT NULL DEFAULT 0,
    total_losses INTEGER NOT NULL DEFAULT 0,
    biggest_win DECIMAL(20,2) NOT NULL DEFAULT 0.00,
    biggest_multiplier DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    average_bet DECIMAL(20,2) NOT NULL DEFAULT 0.00,
    win_rate DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    profit_loss DECIMAL(20,2) NOT NULL DEFAULT 0.00,
    last_updated TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL,
    amount DECIMAL(20,2) NOT NULL,
    balance_before DECIMAL(20,2) NOT NULL,
    balance_after DECIMAL(20,2) NOT NULL,
    reference_id VARCHAR(100),
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_transaction_type CHECK (type IN ('BET', 'WIN', 'DEPOSIT', 'WITHDRAWAL', 'REFUND', 'BONUS'))
);

CREATE INDEX IF NOT EXISTS idx_bets_user_id ON bets(user_id);
CREATE INDEX IF NOT EXISTS idx_bets_round_id ON bets(round_id);
CREATE INDEX IF NOT EXISTS idx_bets_placed_at ON bets(placed_at DESC);
CREATE INDEX IF NOT EXISTS idx_bets_result ON bets(result);

CREATE INDEX IF NOT EXISTS idx_game_rounds_started_at ON game_rounds(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_game_rounds_status ON game_rounds(status);
CREATE INDEX IF NOT EXISTS idx_game_rounds_nonce ON game_rounds(nonce);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);

CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type);

CREATE OR REPLACE FUNCTION update_user_statistics()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.result IN ('WIN', 'LOSS') THEN
        INSERT INTO user_statistics (user_id, total_bets, total_wins, total_losses, biggest_win, biggest_multiplier, profit_loss)
        VALUES (
            NEW.user_id,
            1,
            CASE WHEN NEW.result = 'WIN' THEN 1 ELSE 0 END,
            CASE WHEN NEW.result = 'LOSS' THEN 1 ELSE 0 END,
            COALESCE(NEW.payout - NEW.amount, 0),
            COALESCE(NEW.cashout_multiplier, 0),
            COALESCE(NEW.profit, 0)
        )
        ON CONFLICT (user_id) DO UPDATE SET
            total_bets = user_statistics.total_bets + 1,
            total_wins = user_statistics.total_wins + CASE WHEN NEW.result = 'WIN' THEN 1 ELSE 0 END,
            total_losses = user_statistics.total_losses + CASE WHEN NEW.result = 'LOSS' THEN 1 ELSE 0 END,
            biggest_win = GREATEST(user_statistics.biggest_win, COALESCE(NEW.payout - NEW.amount, 0)),
            biggest_multiplier = GREATEST(user_statistics.biggest_multiplier, COALESCE(NEW.cashout_multiplier, 0)),
            profit_loss = user_statistics.profit_loss + COALESCE(NEW.profit, 0),
            last_updated = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_user_statistics
AFTER INSERT OR UPDATE ON bets
FOR EACH ROW
EXECUTE FUNCTION update_user_statistics();

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

INSERT INTO users (id, username, email, balance) VALUES
    ('00000000-0000-0000-0000-000000000001', 'demo_user_1', 'demo1@example.com', 10000.00),
    ('00000000-0000-0000-0000-000000000002', 'demo_user_2', 'demo2@example.com', 5000.00),
    ('00000000-0000-0000-0000-000000000003', 'demo_user_3', 'demo3@example.com', 15000.00)
ON CONFLICT (id) DO NOTHING;

COMMENT ON TABLE users IS 'Stores user account information and balances';
COMMENT ON TABLE game_rounds IS 'Stores each game round with provably fair data';
COMMENT ON TABLE bets IS 'Stores all bets placed by users';
COMMENT ON TABLE user_statistics IS 'Cached aggregated statistics for each user';
COMMENT ON TABLE transactions IS 'Audit trail for all balance changes';
