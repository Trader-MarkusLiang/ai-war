package evotypes

import (
	"time"
)

// Evolution status constants
const (
	StatusCreated   = "created"
	StatusRunning   = "running"
	StatusPaused    = "paused"
	StatusCompleted = "completed"
	StatusStopped   = "stopped"
)

// Iteration status constants
const (
	IterStatusPending    = "pending"
	IterStatusBacktest   = "backtesting"
	IterStatusEvaluating = "evaluating"
	IterStatusOptimizing = "optimizing"
	IterStatusCompleted  = "completed"
	IterStatusFailed     = "failed"
)

// EvolutionConfig defines the configuration for an evolution task
type EvolutionConfig struct {
	UserID               string      `json:"user_id"`
	Name                 string      `json:"name"`
	BaseStrategyID       string      `json:"base_strategy_id"`
	MaxIterations        int         `json:"max_iterations"`
	ConvergenceThreshold int         `json:"convergence_threshold"` // Stop after N iterations without improvement
	FixedParams          FixedParams `json:"fixed_params"`
	EvaluationModel      string      `json:"evaluation_model"` // AI model for evaluation (e.g., "claude-opus")
}

// FixedParams defines the fixed backtest parameters
type FixedParams struct {
	Symbols           []string `json:"symbols"`
	Timeframes        []string `json:"timeframes"`
	StartTS           int64    `json:"start_ts"`
	EndTS             int64    `json:"end_ts"`
	InitialBalance    float64  `json:"initial_balance"`
	FeeBps            float64  `json:"fee_bps"`
	SlippageBps       float64  `json:"slippage_bps"`
	DecisionTimeframe string   `json:"decision_timeframe"`
	DecisionCadence   int      `json:"decision_cadence_nbars"`
	BTCETHLeverage    int      `json:"btc_eth_leverage"`
	AltcoinLeverage   int      `json:"altcoin_leverage"`
	AIModelID         string   `json:"ai_model_id"`
	CacheAI           bool     `json:"cache_ai"`
}

// Evolution represents an evolution task
type Evolution struct {
	ID                   string       `json:"id"`
	UserID               string       `json:"user_id"`
	Name                 string       `json:"name"`
	BaseStrategyID       string       `json:"base_strategy_id"`
	Status               string       `json:"status"`
	CurrentIteration     int          `json:"current_iteration"`
	MaxIterations        int          `json:"max_iterations"`
	ConvergenceThreshold int          `json:"convergence_threshold"`
	BestVersion          int          `json:"best_version"`
	BestReturn           float64      `json:"best_return"`
	BestDrawdown         float64      `json:"best_drawdown"`
	Config               string       `json:"config"` // JSON string of EvolutionConfig
	CurrentBacktestID    string       `json:"current_backtest_id,omitempty"`
	BacktestProgress     float64      `json:"backtest_progress"`
	// Real-time backtest metrics for display
	CurrentEquity    float64 `json:"current_equity,omitempty"`
	CurrentReturnPct float64 `json:"current_return_pct,omitempty"`
	CurrentDrawdown  float64 `json:"current_drawdown,omitempty"`
	AIModelName      string  `json:"ai_model_name,omitempty"` // AI model name for display
	Iterations       []*Iteration `json:"iterations,omitempty"` // Recent iterations for display
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
}

// Iteration represents a single iteration in the evolution process
type Iteration struct {
	ID             int       `json:"id"`
	EvolutionID    string    `json:"evolution_id"`
	Version        int       `json:"version"`
	StrategyID     string    `json:"strategy_id"`
	BacktestRunID  string    `json:"backtest_run_id"`
	Status         string    `json:"status"`
	Metrics        *Metrics  `json:"metrics,omitempty"`
	EvalReport     string    `json:"evaluation_report,omitempty"` // JSON string
	ChangesSummary string    `json:"changes_summary,omitempty"`
	PromptBefore   string    `json:"prompt_before,omitempty"`
	PromptAfter    string    `json:"prompt_after,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// Metrics holds backtest performance metrics
type Metrics struct {
	TotalReturn float64 `json:"total_return"`
	MaxDrawdown float64 `json:"max_drawdown"`
	WinRate     float64 `json:"win_rate"`
	SharpeRatio float64 `json:"sharpe_ratio"`
	Trades      int     `json:"trades"`
}

// EvaluationReport holds the AI evaluation results
type EvaluationReport struct {
	Strengths   []string `json:"strengths"`
	Weaknesses  []string `json:"weaknesses"`
	Suggestions []string `json:"suggestions"`
	RawResponse string   `json:"raw_response,omitempty"`
}

// OptimizationResult holds the prompt optimization results
type OptimizationResult struct {
	Changes        []string `json:"changes"`
	NewPrompt      string   `json:"new_prompt"`
	ExpectedEffect string   `json:"expected_effect"`
	RawResponse    string   `json:"raw_response,omitempty"`
}

// DecisionSample represents a sampled trading decision for analysis
type DecisionSample struct {
	Timestamp  int64   `json:"timestamp"`
	Symbol     string  `json:"symbol"`
	Action     string  `json:"action"`
	Reasoning  string  `json:"reasoning"`
	PnL        float64 `json:"pnl"`
	IsKeyEvent bool    `json:"is_key_event"` // Large profit or loss
}

// IterationDetail extends Iteration with detailed information
type IterationDetail struct {
	Iteration
	EvaluationReportParsed *EvaluationReport `json:"evaluation_report_parsed,omitempty"`
	PromptDiff             *PromptDiff       `json:"prompt_diff,omitempty"`
	DecisionSamples        []DecisionSample  `json:"decision_samples,omitempty"`
	EquityCurve            []EquityPoint     `json:"equity_curve,omitempty"`
}

// PromptDiff shows the differences between prompts
type PromptDiff struct {
	Before  string   `json:"before"`
	After   string   `json:"after"`
	Changes []string `json:"changes"`
}

// EquityPoint represents a point on the equity curve
type EquityPoint struct {
	Timestamp int64   `json:"timestamp"`
	Equity    float64 `json:"equity"`
	Return    float64 `json:"return"` // Percentage return
}

// EvolutionStatus provides current status information
type EvolutionStatus struct {
	Evolution        *Evolution   `json:"evolution"`
	CurrentIteration *Iteration   `json:"current_iteration,omitempty"`
	RecentIterations []*Iteration `json:"recent_iterations,omitempty"`
	IsConverged      bool         `json:"is_converged"`
	ConvergeReason   string       `json:"converge_reason,omitempty"`
}
