package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// BaselineStrategy represents a saved baseline strategy configuration
type BaselineStrategy struct {
	ID              string         `json:"id"`
	UserID          string         `json:"user_id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	Config          BaselineConfig `json:"config"`
	IsSystemDefault bool           `json:"is_system_default"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// BaselineStrategyPerformance tracks historical performance of a baseline strategy
type BaselineStrategyPerformance struct {
	ID                 int       `json:"id"`
	BaselineStrategyID string    `json:"baseline_strategy_id"`
	RunID              string    `json:"run_id"`
	Symbols            []string  `json:"symbols"`
	Timeframe          string    `json:"timeframe"`
	StartTS            int64     `json:"start_ts"`
	EndTS              int64     `json:"end_ts"`
	InitialBalance     float64   `json:"initial_balance"`
	FinalEquity        float64   `json:"final_equity"`
	TotalReturnPct     float64   `json:"total_return_pct"`
	MaxDrawdownPct     float64   `json:"max_drawdown_pct"`
	SharpeRatio        float64   `json:"sharpe_ratio"`
	WinRate            float64   `json:"win_rate"`
	TotalTrades        int       `json:"total_trades"`
	CreatedAt          time.Time `json:"created_at"`
}

// AggregatedStats contains aggregated performance statistics for a baseline strategy
type AggregatedStats struct {
	TotalRuns      int     `json:"total_runs"`
	AvgReturnPct   float64 `json:"avg_return_pct"`
	AvgDrawdownPct float64 `json:"avg_drawdown_pct"`
	AvgSharpeRatio float64 `json:"avg_sharpe_ratio"`
	AvgWinRate     float64 `json:"avg_win_rate"`
	BestReturnPct  float64 `json:"best_return_pct"`
	WorstReturnPct float64 `json:"worst_return_pct"`
}

// BaselineStrategyWithStats combines strategy with its performance stats
type BaselineStrategyWithStats struct {
	BaselineStrategy
	Stats *AggregatedStats `json:"stats,omitempty"`
}

// BaselineStrategyStore handles baseline strategy persistence
type BaselineStrategyStore struct {
	db *sql.DB
}

// NewBaselineStrategyStore creates a new baseline strategy store
func NewBaselineStrategyStore(db *sql.DB) *BaselineStrategyStore {
	return &BaselineStrategyStore{db: db}
}

// Create creates a new baseline strategy
func (s *BaselineStrategyStore) Create(strategy *BaselineStrategy) error {
	configJSON, err := json.Marshal(strategy.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO baseline_strategies (id, user_id, name, description, config_json, is_system_default)
		VALUES (?, ?, ?, ?, ?, ?)
	`, strategy.ID, strategy.UserID, strategy.Name, strategy.Description, string(configJSON), strategy.IsSystemDefault)

	return err
}

// Update updates an existing baseline strategy
func (s *BaselineStrategyStore) Update(strategy *BaselineStrategy) error {
	configJSON, err := json.Marshal(strategy.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	result, err := s.db.Exec(`
		UPDATE baseline_strategies
		SET name = ?, description = ?, config_json = ?
		WHERE id = ? AND user_id = ?
	`, strategy.Name, strategy.Description, string(configJSON), strategy.ID, strategy.UserID)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("baseline strategy not found or access denied")
	}

	return nil
}

// Delete deletes a baseline strategy
func (s *BaselineStrategyStore) Delete(userID, id string) error {
	result, err := s.db.Exec(`
		DELETE FROM baseline_strategies
		WHERE id = ? AND user_id = ?
	`, id, userID)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("baseline strategy not found or access denied")
	}

	return nil
}

// Get retrieves a single baseline strategy by ID
func (s *BaselineStrategyStore) Get(userID, id string) (*BaselineStrategy, error) {
	var strategy BaselineStrategy
	var configJSON string

	err := s.db.QueryRow(`
		SELECT id, user_id, name, description, config_json, is_system_default, created_at, updated_at
		FROM baseline_strategies
		WHERE id = ? AND (user_id = ? OR is_system_default = 1)
	`, id, userID).Scan(
		&strategy.ID,
		&strategy.UserID,
		&strategy.Name,
		&strategy.Description,
		&configJSON,
		&strategy.IsSystemDefault,
		&strategy.CreatedAt,
		&strategy.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("baseline strategy not found")
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(configJSON), &strategy.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &strategy, nil
}

// GetByName retrieves a baseline strategy by name
func (s *BaselineStrategyStore) GetByName(userID, name string) (*BaselineStrategy, error) {
	var strategy BaselineStrategy
	var configJSON string

	err := s.db.QueryRow(`
		SELECT id, user_id, name, description, config_json, is_system_default, created_at, updated_at
		FROM baseline_strategies
		WHERE name = ? AND (user_id = ? OR is_system_default = 1)
	`, name, userID).Scan(
		&strategy.ID,
		&strategy.UserID,
		&strategy.Name,
		&strategy.Description,
		&configJSON,
		&strategy.IsSystemDefault,
		&strategy.CreatedAt,
		&strategy.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("baseline strategy not found")
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(configJSON), &strategy.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &strategy, nil
}

// List retrieves all baseline strategies for a user (including system defaults)
func (s *BaselineStrategyStore) List(userID string) ([]*BaselineStrategy, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, name, description, config_json, is_system_default, created_at, updated_at
		FROM baseline_strategies
		WHERE user_id = ? OR is_system_default = 1
		ORDER BY is_system_default DESC, created_at DESC
	`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var strategies []*BaselineStrategy
	for rows.Next() {
		var strategy BaselineStrategy
		var configJSON string

		if err := rows.Scan(
			&strategy.ID,
			&strategy.UserID,
			&strategy.Name,
			&strategy.Description,
			&configJSON,
			&strategy.IsSystemDefault,
			&strategy.CreatedAt,
			&strategy.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(configJSON), &strategy.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		strategies = append(strategies, &strategy)
	}

	return strategies, rows.Err()
}

// SavePerformance saves performance metrics for a baseline strategy after a backtest run
func (s *BaselineStrategyStore) SavePerformance(perf *BaselineStrategyPerformance) error {
	symbolsJSON, err := json.Marshal(perf.Symbols)
	if err != nil {
		return fmt.Errorf("failed to marshal symbols: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO baseline_strategy_performance (
			baseline_strategy_id, run_id, symbols, timeframe, start_ts, end_ts,
			initial_balance, final_equity, total_return_pct, max_drawdown_pct,
			sharpe_ratio, win_rate, total_trades
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, perf.BaselineStrategyID, perf.RunID, string(symbolsJSON), perf.Timeframe,
		perf.StartTS, perf.EndTS, perf.InitialBalance, perf.FinalEquity,
		perf.TotalReturnPct, perf.MaxDrawdownPct, perf.SharpeRatio,
		perf.WinRate, perf.TotalTrades)

	return err
}

// GetPerformanceHistory retrieves performance history for a baseline strategy
func (s *BaselineStrategyStore) GetPerformanceHistory(baselineStrategyID string, limit int) ([]*BaselineStrategyPerformance, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.Query(`
		SELECT id, baseline_strategy_id, run_id, symbols, timeframe, start_ts, end_ts,
			initial_balance, final_equity, total_return_pct, max_drawdown_pct,
			sharpe_ratio, win_rate, total_trades, created_at
		FROM baseline_strategy_performance
		WHERE baseline_strategy_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, baselineStrategyID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var performances []*BaselineStrategyPerformance
	for rows.Next() {
		var perf BaselineStrategyPerformance
		var symbolsJSON string

		if err := rows.Scan(
			&perf.ID,
			&perf.BaselineStrategyID,
			&perf.RunID,
			&symbolsJSON,
			&perf.Timeframe,
			&perf.StartTS,
			&perf.EndTS,
			&perf.InitialBalance,
			&perf.FinalEquity,
			&perf.TotalReturnPct,
			&perf.MaxDrawdownPct,
			&perf.SharpeRatio,
			&perf.WinRate,
			&perf.TotalTrades,
			&perf.CreatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(symbolsJSON), &perf.Symbols); err != nil {
			return nil, fmt.Errorf("failed to unmarshal symbols: %w", err)
		}

		performances = append(performances, &perf)
	}

	return performances, rows.Err()
}

// GetAggregatedStats calculates aggregated performance statistics for a baseline strategy
func (s *BaselineStrategyStore) GetAggregatedStats(baselineStrategyID string) (*AggregatedStats, error) {
	var stats AggregatedStats
	var totalRuns sql.NullInt64
	var avgReturn, avgDrawdown, avgSharpe, avgWinRate sql.NullFloat64
	var bestReturn, worstReturn sql.NullFloat64

	err := s.db.QueryRow(`
		SELECT
			COUNT(*) as total_runs,
			AVG(total_return_pct) as avg_return_pct,
			AVG(max_drawdown_pct) as avg_drawdown_pct,
			AVG(sharpe_ratio) as avg_sharpe_ratio,
			AVG(win_rate) as avg_win_rate,
			MAX(total_return_pct) as best_return_pct,
			MIN(total_return_pct) as worst_return_pct
		FROM baseline_strategy_performance
		WHERE baseline_strategy_id = ?
	`, baselineStrategyID).Scan(
		&totalRuns,
		&avgReturn,
		&avgDrawdown,
		&avgSharpe,
		&avgWinRate,
		&bestReturn,
		&worstReturn,
	)

	if err != nil {
		return nil, err
	}

	if totalRuns.Valid {
		stats.TotalRuns = int(totalRuns.Int64)
	}
	if avgReturn.Valid {
		stats.AvgReturnPct = avgReturn.Float64
	}
	if avgDrawdown.Valid {
		stats.AvgDrawdownPct = avgDrawdown.Float64
	}
	if avgSharpe.Valid {
		stats.AvgSharpeRatio = avgSharpe.Float64
	}
	if avgWinRate.Valid {
		stats.AvgWinRate = avgWinRate.Float64
	}
	if bestReturn.Valid {
		stats.BestReturnPct = bestReturn.Float64
	}
	if worstReturn.Valid {
		stats.WorstReturnPct = worstReturn.Float64
	}

	return &stats, nil
}

// ListWithPerformance retrieves all baseline strategies with their performance stats
func (s *BaselineStrategyStore) ListWithPerformance(userID string) ([]*BaselineStrategyWithStats, error) {
	strategies, err := s.List(userID)
	if err != nil {
		return nil, err
	}

	var result []*BaselineStrategyWithStats
	for _, strategy := range strategies {
		stats, err := s.GetAggregatedStats(strategy.ID)
		if err != nil {
			// If no performance data exists, stats will be nil
			stats = nil
		}

		result = append(result, &BaselineStrategyWithStats{
			BaselineStrategy: *strategy,
			Stats:            stats,
		})
	}

	return result, nil
}
