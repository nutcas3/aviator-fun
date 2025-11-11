DROP INDEX IF EXISTS idx_dice_games_win;
DROP INDEX IF EXISTS idx_dice_games_created_at;
DROP INDEX IF EXISTS idx_dice_games_user_id;

DROP INDEX IF EXISTS idx_plinko_games_created_at;
DROP INDEX IF EXISTS idx_plinko_games_user_id;

DROP INDEX IF EXISTS idx_mines_games_status;
DROP INDEX IF EXISTS idx_mines_games_created_at;
DROP INDEX IF EXISTS idx_mines_games_user_id;

DROP INDEX IF EXISTS idx_bets_game_type;
DROP INDEX IF EXISTS idx_game_rounds_game_type;

DROP TABLE IF EXISTS dice_games;
DROP TABLE IF EXISTS plinko_games;
DROP TABLE IF EXISTS mines_games;

ALTER TABLE bets DROP COLUMN IF EXISTS game_type;
ALTER TABLE game_rounds DROP COLUMN IF EXISTS game_type;
