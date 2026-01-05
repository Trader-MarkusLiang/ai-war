package autoevolver

import (
	"nofx/evotypes"
	"nofx/logger"
)

// getBestReturn gets the current best return
func (e *AutoEvolver) getBestReturn() float64 {
	evolution, err := e.store.Evolution().Get(e.config.UserID, e.evolutionID)
	if err != nil {
		return 0
	}
	return evolution.BestReturn
}

// getBestDrawdown gets the current best drawdown
func (e *AutoEvolver) getBestDrawdown() float64 {
	evolution, err := e.store.Evolution().Get(e.config.UserID, e.evolutionID)
	if err != nil {
		return 100 // Return high drawdown as default
	}
	// If best_drawdown is 0 (old data), try to get from best iteration
	if evolution.BestDrawdown == 0 && evolution.BestVersion > 0 {
		iter, err := e.store.Evolution().GetIteration(e.evolutionID, evolution.BestVersion)
		if err == nil && iter != nil && iter.Metrics != nil {
			return iter.Metrics.MaxDrawdown
		}
	}
	return evolution.BestDrawdown
}

// updateBestVersion updates the best version, return and drawdown
func (e *AutoEvolver) updateBestVersion(version int, totalReturn, maxDrawdown float64) {
	if err := e.store.Evolution().UpdateBestVersion(e.evolutionID, version, totalReturn, maxDrawdown); err != nil {
		logger.Errorf("Failed to update best version: %v", err)
	}
}

// getBestStrategyID gets the strategy ID of the best performing iteration
func (e *AutoEvolver) getBestStrategyID() string {
	evolution, err := e.store.Evolution().Get(e.config.UserID, e.evolutionID)
	if err != nil || evolution.BestVersion == 0 {
		return ""
	}

	// Get the iteration record for the best version
	iter, err := e.store.Evolution().GetIteration(e.evolutionID, evolution.BestVersion)
	if err != nil || iter == nil {
		return ""
	}

	return iter.StrategyID
}

// getBestIteration gets the best performing iteration record
func (e *AutoEvolver) getBestIteration() *evotypes.Iteration {
	evolution, err := e.store.Evolution().Get(e.config.UserID, e.evolutionID)
	if err != nil || evolution.BestVersion == 0 {
		return nil
	}

	iter, err := e.store.Evolution().GetIteration(e.evolutionID, evolution.BestVersion)
	if err != nil || iter == nil {
		return nil
	}

	return iter
}
