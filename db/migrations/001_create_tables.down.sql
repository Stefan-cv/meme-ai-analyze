-- ============================================================
-- Meme AI Analyze - Rollback Migration 001
-- ============================================================

DROP TRIGGER IF EXISTS trg_creator_token_update ON creator_tokens;
DROP FUNCTION IF EXISTS update_creator_stats();

DROP TABLE IF EXISTS indexer_state CASCADE;
DROP TABLE IF EXISTS scan_logs CASCADE;
DROP TABLE IF EXISTS creator_tokens CASCADE;
DROP TABLE IF EXISTS creator_records CASCADE;
DROP TABLE IF EXISTS risk_reports CASCADE;
