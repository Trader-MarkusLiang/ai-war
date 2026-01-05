package store

import (
	"database/sql"
	"fmt"
	"time"

	"nofx/evotypes"
)

// EvolutionStore manages evolution task storage
type EvolutionStore struct {
	db *sql.DB
}

// initTables creates the evolution-related tables
func (s *EvolutionStore) initTables() error {
	// Create evolutions table
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS evolutions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			base_strategy_id TEXT NOT NULL,
			status TEXT DEFAULT 'created',
			current_iteration INTEGER DEFAULT 0,
			max_iterations INTEGER DEFAULT 10,
			convergence_threshold INTEGER DEFAULT 3,
			best_version INTEGER DEFAULT 0,
			best_return REAL DEFAULT 0,
			config TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("create evolutions table: %w", err)
	}

	// Create evolution_iterations table
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS evolution_iterations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			evolution_id TEXT NOT NULL,
			version INTEGER NOT NULL,
			strategy_id TEXT NOT NULL,
			backtest_run_id TEXT,
			status TEXT DEFAULT 'pending',
			total_return REAL,
			max_drawdown REAL,
			win_rate REAL,
			sharpe_ratio REAL,
			trades INTEGER,
			evaluation_report TEXT,
			changes_summary TEXT,
			prompt_before TEXT,
			prompt_after TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (evolution_id) REFERENCES evolutions(id),
			UNIQUE(evolution_id, version)
		)
	`)
	if err != nil {
		return fmt.Errorf("create evolution_iterations table: %w", err)
	}

	// Create indexes
	_, _ = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_evolutions_user ON evolutions(user_id)`)
	_, _ = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_evolutions_status ON evolutions(status)`)
	_, _ = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_iterations_evolution ON evolution_iterations(evolution_id)`)

	// Migration: add best_drawdown column if not exists
	_, _ = s.db.Exec(`ALTER TABLE evolutions ADD COLUMN best_drawdown REAL DEFAULT 0`)

	// Create trigger for updated_at
	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS update_evolutions_updated_at
		AFTER UPDATE ON evolutions
		BEGIN
			UPDATE evolutions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END
	`)

	return err
}

// Create creates a new evolution task
func (s *EvolutionStore) Create(evo *evotypes.Evolution) error {
	_, err := s.db.Exec(`
		INSERT INTO evolutions (id, user_id, name, base_strategy_id, status,
			current_iteration, max_iterations, convergence_threshold,
			best_version, best_return, config)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, evo.ID, evo.UserID, evo.Name, evo.BaseStrategyID, evo.Status,
		evo.CurrentIteration, evo.MaxIterations, evo.ConvergenceThreshold,
		evo.BestVersion, evo.BestReturn, evo.Config)
	return err
}

// Get retrieves an evolution task by ID
func (s *EvolutionStore) Get(userID, evolutionID string) (*evotypes.Evolution, error) {
	var evo evotypes.Evolution
	var createdAt, updatedAt string

	err := s.db.QueryRow(`
		SELECT id, user_id, name, base_strategy_id, status, current_iteration,
			max_iterations, convergence_threshold, best_version, best_return,
			COALESCE(best_drawdown, 0), config, created_at, updated_at
		FROM evolutions
		WHERE id = ? AND user_id = ?
	`, evolutionID, userID).Scan(
		&evo.ID, &evo.UserID, &evo.Name, &evo.BaseStrategyID, &evo.Status,
		&evo.CurrentIteration, &evo.MaxIterations, &evo.ConvergenceThreshold,
		&evo.BestVersion, &evo.BestReturn, &evo.BestDrawdown, &evo.Config, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	evo.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	evo.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

	return &evo, nil
}

// List retrieves all evolution tasks for a user
func (s *EvolutionStore) List(userID string) ([]*evotypes.Evolution, error) {
	rows, err := s.db.Query(`
		SELECT id, user_id, name, base_strategy_id, status, current_iteration,
			max_iterations, convergence_threshold, best_version, best_return,
			COALESCE(best_drawdown, 0), config, created_at, updated_at
		FROM evolutions
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var evolutions []*evotypes.Evolution
	for rows.Next() {
		var evo evotypes.Evolution
		var createdAt, updatedAt string

		err := rows.Scan(
			&evo.ID, &evo.UserID, &evo.Name, &evo.BaseStrategyID, &evo.Status,
			&evo.CurrentIteration, &evo.MaxIterations, &evo.ConvergenceThreshold,
			&evo.BestVersion, &evo.BestReturn, &evo.BestDrawdown, &evo.Config, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		evo.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		evo.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)

		evolutions = append(evolutions, &evo)
	}

	return evolutions, nil
}

// UpdateStatus updates the status of an evolution task
func (s *EvolutionStore) UpdateStatus(evolutionID, status string) error {
	_, err := s.db.Exec(`
		UPDATE evolutions SET status = ? WHERE id = ?
	`, status, evolutionID)
	return err
}

// UpdateProgress updates the current iteration and best version
func (s *EvolutionStore) UpdateProgress(evolutionID string, currentIter, bestVersion int, bestReturn float64) error {
	_, err := s.db.Exec(`
		UPDATE evolutions
		SET current_iteration = ?, best_version = ?, best_return = ?
		WHERE id = ?
	`, currentIter, bestVersion, bestReturn, evolutionID)
	return err
}

// UpdateCurrentIteration updates the current iteration
func (s *EvolutionStore) UpdateCurrentIteration(evolutionID string, currentIter int) error {
	_, err := s.db.Exec(`
		UPDATE evolutions SET current_iteration = ? WHERE id = ?
	`, currentIter, evolutionID)
	return err
}

// UpdateBestVersion updates the best version, return and drawdown
func (s *EvolutionStore) UpdateBestVersion(evolutionID string, bestVersion int, bestReturn, bestDrawdown float64) error {
	_, err := s.db.Exec(`
		UPDATE evolutions SET best_version = ?, best_return = ?, best_drawdown = ? WHERE id = ?
	`, bestVersion, bestReturn, bestDrawdown, evolutionID)
	return err
}

// UpdateBaseStrategy updates the base_strategy_id for next iteration
func (s *EvolutionStore) UpdateBaseStrategy(evolutionID, strategyID string) error {
	_, err := s.db.Exec(`
		UPDATE evolutions SET base_strategy_id = ? WHERE id = ?
	`, strategyID, evolutionID)
	return err
}

// CreateIteration creates a new iteration record
func (s *EvolutionStore) CreateIteration(iter *evotypes.Iteration) error {
	var totalReturn, maxDrawdown, winRate, sharpeRatio sql.NullFloat64
	var trades sql.NullInt64

	if iter.Metrics != nil {
		totalReturn = sql.NullFloat64{Float64: iter.Metrics.TotalReturn, Valid: true}
		maxDrawdown = sql.NullFloat64{Float64: iter.Metrics.MaxDrawdown, Valid: true}
		winRate = sql.NullFloat64{Float64: iter.Metrics.WinRate, Valid: true}
		sharpeRatio = sql.NullFloat64{Float64: iter.Metrics.SharpeRatio, Valid: true}
		trades = sql.NullInt64{Int64: int64(iter.Metrics.Trades), Valid: true}
	}

	_, err := s.db.Exec(`
		INSERT INTO evolution_iterations (
			evolution_id, version, strategy_id, backtest_run_id, status,
			total_return, max_drawdown, win_rate, sharpe_ratio, trades,
			evaluation_report, changes_summary, prompt_before, prompt_after
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, iter.EvolutionID, iter.Version, iter.StrategyID, iter.BacktestRunID, iter.Status,
		totalReturn, maxDrawdown, winRate, sharpeRatio, trades,
		iter.EvalReport, iter.ChangesSummary, iter.PromptBefore, iter.PromptAfter)

	return err
}

// GetIterations retrieves all iterations for an evolution task
func (s *EvolutionStore) GetIterations(evolutionID string) ([]*evotypes.Iteration, error) {
	rows, err := s.db.Query(`
		SELECT id, evolution_id, version, strategy_id, backtest_run_id, status,
			total_return, max_drawdown, win_rate, sharpe_ratio, trades,
			evaluation_report, changes_summary, prompt_before, prompt_after, created_at
		FROM evolution_iterations
		WHERE evolution_id = ?
		ORDER BY version ASC
	`, evolutionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var iterations []*evotypes.Iteration
	for rows.Next() {
		iter, err := s.scanIteration(rows)
		if err != nil {
			return nil, err
		}
		iterations = append(iterations, iter)
	}

	return iterations, nil
}

// scanIteration scans a row into an Iteration struct
func (s *EvolutionStore) scanIteration(scanner interface {
	Scan(dest ...interface{}) error
}) (*evotypes.Iteration, error) {
	var iter evotypes.Iteration
	var totalReturn, maxDrawdown, winRate, sharpeRatio sql.NullFloat64
	var trades sql.NullInt64
	var createdAt string
	var evalReport, changesSummary, promptBefore, promptAfter sql.NullString

	err := scanner.Scan(
		&iter.ID, &iter.EvolutionID, &iter.Version, &iter.StrategyID,
		&iter.BacktestRunID, &iter.Status,
		&totalReturn, &maxDrawdown, &winRate, &sharpeRatio, &trades,
		&evalReport, &changesSummary,
		&promptBefore, &promptAfter, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	// Handle nullable string fields
	iter.EvalReport = evalReport.String
	iter.ChangesSummary = changesSummary.String
	iter.PromptBefore = promptBefore.String
	iter.PromptAfter = promptAfter.String

	// Parse metrics if available
	if totalReturn.Valid {
		iter.Metrics = &evotypes.Metrics{
			TotalReturn: totalReturn.Float64,
			MaxDrawdown: maxDrawdown.Float64,
			WinRate:     winRate.Float64,
			SharpeRatio: sharpeRatio.Float64,
			Trades:      int(trades.Int64),
		}
	}

	iter.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)

	return &iter, nil
}

// GetIteration retrieves a specific iteration by version
func (s *EvolutionStore) GetIteration(evolutionID string, version int) (*evotypes.Iteration, error) {
	row := s.db.QueryRow(`
		SELECT id, evolution_id, version, strategy_id, backtest_run_id, status,
			total_return, max_drawdown, win_rate, sharpe_ratio, trades,
			evaluation_report, changes_summary, prompt_before, prompt_after, created_at
		FROM evolution_iterations
		WHERE evolution_id = ? AND version = ?
	`, evolutionID, version)

	return s.scanIteration(row)
}

// UpdateIterationStatus updates the status of an iteration
func (s *EvolutionStore) UpdateIterationStatus(evolutionID string, version int, status string) error {
	_, err := s.db.Exec(`
		UPDATE evolution_iterations
		SET status = ?
		WHERE evolution_id = ? AND version = ?
	`, status, evolutionID, version)
	return err
}

// UpdateIterationMetrics updates the metrics of an iteration
func (s *EvolutionStore) UpdateIterationMetrics(evolutionID string, version int, metrics *evotypes.Metrics) error {
	_, err := s.db.Exec(`
		UPDATE evolution_iterations
		SET total_return = ?, max_drawdown = ?, win_rate = ?, sharpe_ratio = ?, trades = ?
		WHERE evolution_id = ? AND version = ?
	`, metrics.TotalReturn, metrics.MaxDrawdown, metrics.WinRate, metrics.SharpeRatio, metrics.Trades,
		evolutionID, version)
	return err
}

// UpdateIterationEvaluation updates the evaluation report of an iteration
func (s *EvolutionStore) UpdateIterationEvaluation(evolutionID string, version int, evalReport, changesSummary string) error {
	_, err := s.db.Exec(`
		UPDATE evolution_iterations
		SET evaluation_report = ?, changes_summary = ?
		WHERE evolution_id = ? AND version = ?
	`, evalReport, changesSummary, evolutionID, version)
	return err
}

// UpdateIterationPrompts updates the before/after prompts of an iteration
func (s *EvolutionStore) UpdateIterationPrompts(evolutionID string, version int, promptBefore, promptAfter string) error {
	_, err := s.db.Exec(`
		UPDATE evolution_iterations
		SET prompt_before = ?, prompt_after = ?
		WHERE evolution_id = ? AND version = ?
	`, promptBefore, promptAfter, evolutionID, version)
	return err
}

// UpdateIterationComplete updates all fields when iteration completes
func (s *EvolutionStore) UpdateIterationComplete(evolutionID string, version int, metrics *evotypes.Metrics, evalReport, changesSummary, promptAfter string) error {
	_, err := s.db.Exec(`
		UPDATE evolution_iterations
		SET status = 'completed',
			total_return = ?, max_drawdown = ?, win_rate = ?, sharpe_ratio = ?, trades = ?,
			evaluation_report = ?, changes_summary = ?, prompt_after = ?
		WHERE evolution_id = ? AND version = ?
	`, metrics.TotalReturn, metrics.MaxDrawdown, metrics.WinRate, metrics.SharpeRatio, metrics.Trades,
		evalReport, changesSummary, promptAfter, evolutionID, version)
	return err
}

// Delete removes an evolution and its iterations from the database
func (s *EvolutionStore) Delete(evolutionID string) error {
	// Delete iterations first
	_, err := s.db.Exec(`DELETE FROM evolution_iterations WHERE evolution_id = ?`, evolutionID)
	if err != nil {
		return err
	}
	// Delete evolution
	_, err = s.db.Exec(`DELETE FROM evolutions WHERE id = ?`, evolutionID)
	return err
}

// ResetRunningToPaused resets all running evolutions to paused state
// This is called on server startup to handle evolutions that were interrupted
func (s *EvolutionStore) ResetRunningToPaused() (int64, error) {
	result, err := s.db.Exec(`
		UPDATE evolutions
		SET status = 'paused', updated_at = CURRENT_TIMESTAMP
		WHERE status = 'running'
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
