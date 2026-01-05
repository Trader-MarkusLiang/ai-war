package autoevolver

import (
	"context"

	"nofx/backtest"
	"nofx/logger"
	"nofx/mcp"
	"nofx/store"
)

// AutoEvolver manages the automatic evolution process
type AutoEvolver struct {
	evolutionID string
	config      *EvolutionConfig
	backtestMgr *backtest.Manager
	aiClient    mcp.AIClient
	store       *store.Store
	status      string
	stopChan    chan struct{}
	pauseChan   chan struct{}
	isPaused    bool
}

// NewAutoEvolver creates a new AutoEvolver instance
func NewAutoEvolver(
	evolutionID string,
	config *EvolutionConfig,
	backtestMgr *backtest.Manager,
	aiClient mcp.AIClient,
	st *store.Store,
) *AutoEvolver {
	return &AutoEvolver{
		evolutionID: evolutionID,
		config:      config,
		backtestMgr: backtestMgr,
		aiClient:    aiClient,
		store:       st,
		status:      StatusCreated,
		stopChan:    make(chan struct{}),
		pauseChan:   make(chan struct{}),
		isPaused:    false,
	}
}

// Start begins the evolution process
func (e *AutoEvolver) Start(ctx context.Context) error {
	e.status = StatusRunning
	logger.Infof("Starting evolution %s", e.evolutionID)

	// Get current progress from database to resume from correct iteration
	evolution, err := e.store.Evolution().Get(e.config.UserID, e.evolutionID)
	startVersion := 1

	logger.Infof("Evolution %s: CurrentIteration from DB = %d, err = %v", e.evolutionID, evolution.CurrentIteration, err)

	if err == nil && evolution.CurrentIteration > 0 {
		// Check if the last iteration was completed successfully
		iterations, iterErr := e.store.Evolution().GetIterations(e.evolutionID)
		logger.Infof("Evolution %s: found %d iterations, err = %v", e.evolutionID, len(iterations), iterErr)

		if iterErr == nil && len(iterations) > 0 {
			lastIter := iterations[len(iterations)-1]
			logger.Infof("Evolution %s: lastIter version=%d, status=%s", e.evolutionID, lastIter.Version, lastIter.Status)

			if lastIter.Status == "completed" {
				// Last iteration completed, start from next
				startVersion = evolution.CurrentIteration + 1
			} else {
				// Last iteration failed/incomplete, retry it
				startVersion = evolution.CurrentIteration
				logger.Infof("Evolution %s: retrying failed iteration %d", e.evolutionID, startVersion)
			}
		} else {
			startVersion = evolution.CurrentIteration + 1
		}
		logger.Infof("Evolution %s: resuming from iteration %d", e.evolutionID, startVersion)
	}

	for version := startVersion; version <= e.config.MaxIterations; version++ {
		// Check for stop signal
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-e.stopChan:
			logger.Infof("Evolution %s stopped by user", e.evolutionID)
			return nil
		default:
		}

		// Check for pause signal
		if e.isPaused {
			logger.Infof("Evolution %s paused at version %d", e.evolutionID, version)
			<-e.pauseChan // Wait for resume
			logger.Infof("Evolution %s resumed", e.evolutionID)
		}

		// Run single iteration
		logger.Infof("Evolution %s: starting iteration %d/%d", e.evolutionID, version, e.config.MaxIterations)

		// Update current iteration BEFORE running (so we can resume from this iteration if it fails)
		e.store.Evolution().UpdateCurrentIteration(e.evolutionID, version)

		if err := e.runIteration(ctx, version); err != nil {
			logger.Errorf("Evolution %s iteration %d failed: %v", e.evolutionID, version, err)
			e.store.Evolution().UpdateStatus(e.evolutionID, StatusStopped)
			return err
		}
	}

	logger.Infof("Evolution %s completed all %d iterations", e.evolutionID, e.config.MaxIterations)
	e.status = StatusCompleted
	e.store.Evolution().UpdateStatus(e.evolutionID, StatusCompleted)
	return nil
}

// Pause pauses the evolution process
func (e *AutoEvolver) Pause() error {
	e.isPaused = true
	e.status = StatusPaused
	logger.Infof("Evolution %s paused", e.evolutionID)
	return nil
}

// Resume resumes the evolution process
func (e *AutoEvolver) Resume(ctx context.Context) error {
	e.isPaused = false
	e.status = StatusRunning
	close(e.pauseChan)
	e.pauseChan = make(chan struct{})
	logger.Infof("Evolution %s resumed", e.evolutionID)
	return nil
}

// Stop stops the evolution process
func (e *AutoEvolver) Stop() error {
	e.status = StatusStopped
	close(e.stopChan)
	logger.Infof("Evolution %s stopped", e.evolutionID)
	return nil
}

// GetStatus returns the current status
func (e *AutoEvolver) GetStatus() string {
	return e.status
}
