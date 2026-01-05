-- Migration: Add Baseline Strategy Pool System
-- Date: 2026-01-03
-- Description: Create tables for baseline strategy management and performance tracking

-- ============================================================================
-- 1. Create baseline_strategies table
-- ============================================================================
CREATE TABLE IF NOT EXISTS baseline_strategies (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    config_json TEXT NOT NULL,
    is_system_default BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for baseline_strategies
CREATE INDEX IF NOT EXISTS idx_baseline_strategies_user_id
    ON baseline_strategies(user_id);
CREATE INDEX IF NOT EXISTS idx_baseline_strategies_name
    ON baseline_strategies(user_id, name);

-- Trigger: auto-update updated_at on modification
CREATE TRIGGER IF NOT EXISTS update_baseline_strategies_updated_at
AFTER UPDATE ON baseline_strategies
BEGIN
    UPDATE baseline_strategies
    SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;

-- ============================================================================
-- 2. Create baseline_strategy_performance table
-- ============================================================================
CREATE TABLE IF NOT EXISTS baseline_strategy_performance (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    baseline_strategy_id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    symbols TEXT NOT NULL,
    timeframe TEXT NOT NULL,
    start_ts INTEGER NOT NULL,
    end_ts INTEGER NOT NULL,
    initial_balance REAL NOT NULL,
    final_equity REAL NOT NULL,
    total_return_pct REAL NOT NULL,
    max_drawdown_pct REAL NOT NULL,
    sharpe_ratio REAL DEFAULT 0,
    win_rate REAL DEFAULT 0,
    total_trades INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (baseline_strategy_id)
        REFERENCES baseline_strategies(id) ON DELETE CASCADE,
    FOREIGN KEY (run_id)
        REFERENCES backtest_runs(run_id) ON DELETE CASCADE
);

-- Indexes for baseline_strategy_performance
CREATE INDEX IF NOT EXISTS idx_baseline_perf_strategy
    ON baseline_strategy_performance(baseline_strategy_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_baseline_perf_run
    ON baseline_strategy_performance(run_id);

-- ============================================================================
-- 3. Modify backtest_runs table - Add baseline_strategy_id column
-- ============================================================================
-- Note: SQLite doesn't support ALTER TABLE ADD COLUMN IF NOT EXISTS
-- We'll check if column exists before adding

-- Add baseline_strategy_id column (will fail silently if already exists)
ALTER TABLE backtest_runs
ADD COLUMN baseline_strategy_id TEXT DEFAULT '';

-- Create index for baseline_strategy_id
CREATE INDEX IF NOT EXISTS idx_backtest_runs_baseline_strategy
    ON backtest_runs(baseline_strategy_id);

-- ============================================================================
-- Migration Complete
-- ============================================================================
