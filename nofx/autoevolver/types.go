package autoevolver

import "nofx/evotypes"

// Re-export types from evotypes for backward compatibility
type (
	Evolution          = evotypes.Evolution
	Iteration          = evotypes.Iteration
	Metrics            = evotypes.Metrics
	EvaluationReport   = evotypes.EvaluationReport
	OptimizationResult = evotypes.OptimizationResult
	DecisionSample     = evotypes.DecisionSample
	IterationDetail    = evotypes.IterationDetail
	PromptDiff         = evotypes.PromptDiff
	EquityPoint        = evotypes.EquityPoint
	EvolutionStatus    = evotypes.EvolutionStatus
	EvolutionConfig    = evotypes.EvolutionConfig
	FixedParams        = evotypes.FixedParams
)

// Re-export constants
const (
	StatusCreated   = evotypes.StatusCreated
	StatusRunning   = evotypes.StatusRunning
	StatusPaused    = evotypes.StatusPaused
	StatusCompleted = evotypes.StatusCompleted
	StatusStopped   = evotypes.StatusStopped

	IterStatusPending    = evotypes.IterStatusPending
	IterStatusBacktest   = evotypes.IterStatusBacktest
	IterStatusEvaluating = evotypes.IterStatusEvaluating
	IterStatusOptimizing = evotypes.IterStatusOptimizing
	IterStatusCompleted  = evotypes.IterStatusCompleted
	IterStatusFailed     = evotypes.IterStatusFailed
)
