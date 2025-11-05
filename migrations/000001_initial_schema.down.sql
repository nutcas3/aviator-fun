DROP TRIGGER IF EXISTS trigger_update_user_statistics ON bets;
DROP TRIGGER IF EXISTS trigger_users_updated_at ON users;

DROP FUNCTION IF EXISTS update_user_statistics();
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_transactions_type;
DROP INDEX IF EXISTS idx_transactions_created_at;
DROP INDEX IF EXISTS idx_transactions_user_id;
DROP INDEX IF EXISTS idx_users_is_active;
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_game_rounds_nonce;
DROP INDEX IF EXISTS idx_game_rounds_status;
DROP INDEX IF EXISTS idx_game_rounds_started_at;
DROP INDEX IF EXISTS idx_bets_result;
DROP INDEX IF EXISTS idx_bets_placed_at;
DROP INDEX IF EXISTS idx_bets_round_id;
DROP INDEX IF EXISTS idx_bets_user_id;

DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS user_statistics;
DROP TABLE IF EXISTS bets;
DROP TABLE IF EXISTS game_rounds;
DROP TABLE IF EXISTS users;

DROP EXTENSION IF EXISTS "uuid-ossp";
