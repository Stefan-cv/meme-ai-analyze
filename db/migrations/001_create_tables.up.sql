-- ============================================================
-- Meme AI Analyze - Database Schema Migration
-- Version: 001
-- Description: Initial schema - risk reports, creator tracking,
--              scan logs, indexer state
-- ============================================================

-- ==========================================
-- 1. Risk Reports Table
-- ==========================================
CREATE TABLE IF NOT EXISTS risk_reports (
    id              BIGSERIAL PRIMARY KEY,
    chain_id        INT NOT NULL DEFAULT 56,          -- Chain ID (56=BSC, 1=Ethereum)
    token_address   VARCHAR(42) NOT NULL,
    token_name      VARCHAR(128),
    token_symbol    VARCHAR(32),
    risk_score      DECIMAL(3,1) NOT NULL,            -- 1.0 - 10.0
    risk_level      VARCHAR(16) NOT NULL,             -- low / medium / high / critical
    dimension_json  JSONB NOT NULL,                   -- Six dimension score details
    ai_report       TEXT,                             -- AI-generated analysis text
    report_hash     VARCHAR(128),                     -- IPFS / Arweave content hash
    oracle_tx_hash  VARCHAR(66),                      -- On-chain anchor transaction hash
    creator_address VARCHAR(42),
    scanned_at      BIGINT NOT NULL,                  -- Unix timestamp (seconds)
    created_at      BIGINT NOT NULL DEFAULT (
                        EXTRACT(EPOCH FROM NOW()) * 1000
                    )::BIGINT,                        -- Unix timestamp (milliseconds)
    UNIQUE(chain_id, token_address)
);

-- Indexes for risk_reports
CREATE INDEX IF NOT EXISTS idx_reports_score
    ON risk_reports(chain_id, risk_score);

CREATE INDEX IF NOT EXISTS idx_reports_creator
    ON risk_reports(creator_address);

CREATE INDEX IF NOT EXISTS idx_reports_scanned
    ON risk_reports(scanned_at DESC);

CREATE INDEX IF NOT EXISTS idx_reports_level
    ON risk_reports(chain_id, risk_level);

-- ==========================================
-- 2. Creator Records Table (Founder Profiles)
-- ==========================================
CREATE TABLE IF NOT EXISTS creator_records (
    address             VARCHAR(42) PRIMARY KEY,
    total_tokens        INT NOT NULL DEFAULT 0,
    rugged_tokens       INT NOT NULL DEFAULT 0,
    avg_lifetime_hours  DECIMAL(8,2) DEFAULT 0,
    first_seen_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    risk_label          VARCHAR(32) DEFAULT 'new'     -- clean / suspicious / serial_rugger / new
);

-- Indexes for creator_records
CREATE INDEX IF NOT EXISTS idx_creator_label
    ON creator_records(risk_label);

CREATE INDEX IF NOT EXISTS idx_creator_tokens
    ON creator_records(total_tokens DESC);

CREATE INDEX IF NOT EXISTS idx_creator_rug_rate
    ON creator_records(
        CASE WHEN total_tokens > 0
            THEN rugged_tokens::DECIMAL / total_tokens::DECIMAL
            ELSE 0
        END DESC
    );

-- ==========================================
-- 3. Creator Tokens Table (Founder's Token History)
-- ==========================================
CREATE TABLE IF NOT EXISTS creator_tokens (
    id               BIGSERIAL PRIMARY KEY,
    creator_address  VARCHAR(42) NOT NULL
                     REFERENCES creator_records(address)
                     ON DELETE CASCADE,
    token_address    VARCHAR(42) NOT NULL,
    token_name       VARCHAR(128),
    token_symbol     VARCHAR(32),
    status           VARCHAR(16) DEFAULT 'active',    -- active / rugged / unknown
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rugged_at        TIMESTAMPTZ,                     -- Time when rugged (liquidity drained)
    UNIQUE(token_address)
);

-- Indexes for creator_tokens
CREATE INDEX IF NOT EXISTS idx_ct_creator
    ON creator_tokens(creator_address, status);

CREATE INDEX IF NOT EXISTS idx_ct_created
    ON creator_tokens(creator_address, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ct_status
    ON creator_tokens(status, created_at DESC);

-- ==========================================
-- 4. Scan Logs Table (Usage Tracking)
-- ==========================================
CREATE TABLE IF NOT EXISTS scan_logs (
    id             BIGSERIAL PRIMARY KEY,
    token_address  VARCHAR(42) NOT NULL,
    scanned_by     VARCHAR(42),                       -- Scanner wallet address (nullable)
    risk_score     DECIMAL(3,1),
    scanned_at     BIGINT NOT NULL                    -- Unix timestamp (seconds)
);

-- Indexes for scan_logs
CREATE INDEX IF NOT EXISTS idx_scan_token
    ON scan_logs(token_address, scanned_at DESC);

CREATE INDEX IF NOT EXISTS idx_scan_time
    ON scan_logs(scanned_at DESC);

CREATE INDEX IF NOT EXISTS idx_scan_by
    ON scan_logs(scanned_by, scanned_at DESC);

-- ==========================================
-- 5. Indexer State Table (Resume from last block)
-- ==========================================
CREATE TABLE IF NOT EXISTS indexer_state (
    chain_id        INT PRIMARY KEY,
    last_block      BIGINT NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==========================================
-- Seed initial indexer state for BSC
-- ==========================================
INSERT INTO indexer_state (chain_id, last_block)
VALUES (56, 0)
ON CONFLICT (chain_id) DO NOTHING;

-- ==========================================
-- Helper: Function to update creator stats
-- ==========================================
CREATE OR REPLACE FUNCTION update_creator_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- Recalculate rugged_tokens and avg_lifetime_hours
    UPDATE creator_records
    SET
        rugged_tokens = (
            SELECT COUNT(*)
            FROM creator_tokens
            WHERE creator_address = NEW.creator_address
              AND status = 'rugged'
        ),
        avg_lifetime_hours = (
            SELECT COALESCE(AVG(
                EXTRACT(EPOCH FROM (COALESCE(rugged_at, NOW()) - created_at)) / 3600
            ), 0)
            FROM creator_tokens
            WHERE creator_address = NEW.creator_address
              AND status = 'rugged'
        ),
        last_seen_at = NOW()
    WHERE address = NEW.creator_address;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger: update creator stats when a token status changes
DROP TRIGGER IF EXISTS trg_creator_token_update ON creator_tokens;
CREATE TRIGGER trg_creator_token_update
    AFTER UPDATE OF status ON creator_tokens
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status)
    EXECUTE FUNCTION update_creator_stats();
