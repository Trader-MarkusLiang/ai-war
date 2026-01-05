package autoevolver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nofx/backtest"
	"nofx/evotypes"
	"nofx/logger"
	"nofx/store"

	"github.com/google/uuid"
)

// runIteration executes a single evolution iteration (simplified version)
func (e *AutoEvolver) runIteration(ctx context.Context, version int) error {
	// 1. Get current strategy
	strategy, err := e.store.Strategy().Get(e.config.UserID, e.config.BaseStrategyID)
	if err != nil {
		return fmt.Errorf("failed to get strategy: %w", err)
	}

	// Get strategy config (prompt)
	promptVariant := "baseline"
	if strategy.Config != "" {
		promptVariant = strategy.Config
	}

	// 2. Check if there's an existing iteration with running/completed backtest (resume case)
	existingIter, err := e.store.Evolution().GetIteration(e.evolutionID, version)
	var backtestRunID string
	needReEvaluate := false // Flag to force re-evaluation when backtest is reset

	if err == nil && existingIter != nil && existingIter.BacktestRunID != "" {
		// Check backtest state
		meta, metaErr := e.backtestMgr.LoadMetadata(existingIter.BacktestRunID)
		if metaErr == nil && meta != nil {
			switch meta.State {
			case "running", "paused":
				// Check if backtest is actually running (runner exists)
				statusPayload := e.backtestMgr.Status(existingIter.BacktestRunID)
				if statusPayload != nil {
					// Runner exists, wait for it
					backtestRunID = existingIter.BacktestRunID
					logger.Infof("Evolution %s v%d: resuming existing backtest %s", e.evolutionID, version, backtestRunID)
					goto waitBacktest
				}
				// Runner doesn't exist (service restarted), need to restart backtest
				backtestRunID = existingIter.BacktestRunID
				// Use the prompt that was saved for this iteration
				if existingIter.PromptBefore != "" {
					promptVariant = existingIter.PromptBefore
				}
				logger.Infof("Evolution %s v%d: backtest %s state is %s but runner not found, deleting old data and restarting...", e.evolutionID, version, backtestRunID, meta.State)
				// Delete old backtest data before restarting
				if err := e.backtestMgr.Delete(backtestRunID); err != nil {
					logger.Warnf("Failed to delete old backtest data: %v", err)
				}
				// Mark for re-evaluation since we're restarting the backtest
				needReEvaluate = true
				goto restartBacktest
			case "completed":
				// Backtest already completed
				backtestRunID = existingIter.BacktestRunID
				// Check if evaluation/optimization was already done
				if existingIter.Status == "completed" {
					// Iteration fully completed, skip to next
					logger.Infof("Evolution %s v%d: iteration already completed, skipping", e.evolutionID, version)
					return nil
				}
				// Backtest done but evaluation/optimization not done, proceed to evaluation
				logger.Infof("Evolution %s v%d: backtest %s completed, proceeding to evaluation", e.evolutionID, version, backtestRunID)
				goto evaluateBacktest
			case "stopped":
				// Backtest was stopped, need to restart it with the same ID
				backtestRunID = existingIter.BacktestRunID
				// Use the prompt that was saved for this iteration
				if existingIter.PromptBefore != "" {
					promptVariant = existingIter.PromptBefore
				}
				logger.Infof("Evolution %s v%d: backtest %s was stopped, restarting...", e.evolutionID, version, backtestRunID)
				// Mark for re-evaluation since we're restarting the backtest
				needReEvaluate = true
				goto restartBacktest
			}
		}
	}

	// 3. Create new backtest with readable name format: evo-YYYYMMDD-HHMM-epoch-N
	backtestRunID = fmt.Sprintf("evo-%s-epoch-%d", time.Now().Format("20060102-1504"), version)

restartBacktest:
	{
		backtestConfig := backtest.BacktestConfig{
			RunID:                backtestRunID,
			UserID:               e.config.UserID,
			AIModelID:            e.config.FixedParams.AIModelID,
			StrategyID:           e.config.BaseStrategyID,
			Symbols:              e.config.FixedParams.Symbols,
			Timeframes:           e.config.FixedParams.Timeframes,
			DecisionTimeframe:    e.config.FixedParams.DecisionTimeframe,
			DecisionCadenceNBars: e.config.FixedParams.DecisionCadence,
			StartTS:              e.config.FixedParams.StartTS,
			EndTS:                e.config.FixedParams.EndTS,
			InitialBalance:       e.config.FixedParams.InitialBalance,
			FeeBps:               e.config.FixedParams.FeeBps,
			SlippageBps:          e.config.FixedParams.SlippageBps,
			PromptVariant:        promptVariant,
			CacheAI:              e.config.FixedParams.CacheAI,
		}

		// Load strategy config (indicators, etc.) - use promptVariant which may be from existingIter.PromptBefore
		configToLoad := promptVariant
		if configToLoad == "baseline" || configToLoad == "" {
			configToLoad = strategy.Config
		}
		if configToLoad != "" && configToLoad != "baseline" {
			var strategyConfig store.StrategyConfig
			if err := json.Unmarshal([]byte(configToLoad), &strategyConfig); err == nil {
				backtestConfig.SetLoadedStrategy(&strategyConfig)
				logger.Infof("Evolution %s v%d: loaded strategy config with indicators", e.evolutionID, version)
			} else {
				logger.Warnf("Evolution %s v%d: failed to parse strategy config: %v", e.evolutionID, version, err)
			}
		}

		// Hydrate AI configuration from database
		if err := e.hydrateAIConfig(&backtestConfig); err != nil {
			return fmt.Errorf("failed to hydrate AI config: %w", err)
		}

		logger.Infof("Evolution %s v%d: starting backtest %s", e.evolutionID, version, backtestRunID)

		// Create iteration record with "backtest" status (only if not restarting)
		if existingIter == nil || existingIter.BacktestRunID != backtestRunID {
			iteration := &evotypes.Iteration{
				EvolutionID:   e.evolutionID,
				Version:       version,
				StrategyID:    strategy.ID,
				BacktestRunID: backtestRunID,
				Status:        "backtest",
				PromptBefore:  promptVariant,
			}
			if err := e.store.Evolution().CreateIteration(iteration); err != nil {
				logger.Warnf("Failed to create iteration record: %v", err)
			}
		} else {
			// Update existing iteration status back to "backtest"
			e.store.Evolution().UpdateIterationStatus(e.evolutionID, version, "backtest")
		}

		// Start backtest
		_, err = e.backtestMgr.Start(ctx, backtestConfig)
		if err != nil {
			e.store.Evolution().UpdateIterationStatus(e.evolutionID, version, "failed")
			return fmt.Errorf("backtest start failed: %w", err)
		}
	}

waitBacktest:

	// 5. Wait for backtest to complete
	if err := e.waitForBacktestComplete(ctx, backtestRunID); err != nil {
		return fmt.Errorf("backtest wait failed: %w", err)
	}

evaluateBacktest:

	// 5. Get backtest results
	metrics, err := e.backtestMgr.GetMetrics(backtestRunID)
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	logger.Infof("Evolution %s v%d: backtest completed, return=%.2f%%, drawdown=%.2f%%",
		e.evolutionID, version, metrics.TotalReturnPct, metrics.MaxDrawdownPct)

	// 6. Get trades for analysis
	trades, _ := e.backtestMgr.LoadTrades(backtestRunID, 100)

	// 7. AI Evaluation - update status
	if needReEvaluate {
		logger.Infof("Evolution %s v%d: backtest was reset, forcing re-evaluation and re-optimization", e.evolutionID, version)
	}
	e.store.Evolution().UpdateIterationStatus(e.evolutionID, version, "evaluating")
	logger.Infof("Evolution %s v%d: running AI evaluation...", e.evolutionID, version)
	analyzer := NewAnalyzer(e.aiClient)
	analysisInput := &AnalysisInput{
		Metrics:       metrics,
		CurrentPrompt: promptVariant,
		Trades:        trades,
	}
	evaluation, err := analyzer.Analyze(analysisInput)
	if err != nil {
		logger.Warnf("AI evaluation failed: %v", err)
	}

	// 8. Get iteration history for optimization context
	iterHistory := e.getIterationHistory()

	// 9. AI Optimization - update status
	e.store.Evolution().UpdateIterationStatus(e.evolutionID, version, "optimizing")

	// Check if current epoch is better than best (using same criteria as improvement check)
	currentBestReturn := e.getBestReturn()
	currentBestDrawdown := e.getBestDrawdown()
	returnDiff := metrics.TotalReturnPct - currentBestReturn
	drawdownImprovement := currentBestDrawdown - metrics.MaxDrawdownPct
	isCurrentBetter := returnDiff > 0 || (returnDiff >= -3.0 && drawdownImprovement >= 5.0)

	// Prepare optimization input with comparison data
	bestIter := e.getBestIteration()
	optimInput := &OptimizationInput{
		EvaluationReport: evaluation,
		IterationHistory: iterHistory,
		// Current epoch data
		CurrentMetrics:  metrics,
		CurrentTrades:   trades,
		CurrentVersion:  version,
		IsCurrentBest:   isCurrentBetter || bestIter == nil,
	}

	// If current is best or no best exists, optimize based on current
	if isCurrentBetter || bestIter == nil {
		optimInput.CurrentPrompt = promptVariant
		logger.Infof("Evolution %s v%d: current epoch is best, optimizing based on current", e.evolutionID, version)
	} else {
		// Current is not best - provide both epochs for comparison analysis
		logger.Infof("Evolution %s v%d: current epoch not best, providing comparison with best epoch v%d",
			e.evolutionID, version, bestIter.Version)

		// Use best epoch's prompt as base for optimization
		if bestIter.PromptBefore != "" {
			optimInput.CurrentPrompt = bestIter.PromptBefore
			optimInput.BestPrompt = bestIter.PromptBefore
		} else {
			optimInput.CurrentPrompt = promptVariant
		}
		optimInput.BestVersion = bestIter.Version

		// Load best epoch's trades for comparison
		bestTrades, err := e.backtestMgr.LoadTrades(bestIter.BacktestRunID, 100)
		if err == nil {
			optimInput.BestTrades = bestTrades
		}

		// Load best epoch's metrics for comparison
		bestMetrics, err := e.backtestMgr.GetMetrics(bestIter.BacktestRunID)
		if err == nil && bestMetrics != nil {
			optimInput.BestMetrics = bestMetrics
		}
	}

	logger.Infof("Evolution %s v%d: running AI optimization...", e.evolutionID, version)
	optimizer := NewOptimizer(e.aiClient)
	optimization, err := optimizer.Optimize(optimInput)
	if err != nil {
		logger.Warnf("AI optimization failed: %v", err)
		optimization = &evotypes.OptimizationResult{
			NewPrompt:      promptVariant,
			ExpectedEffect: "Optimization failed, keeping original",
		}
	}

	// 10. Update iteration record with results
	evalJSON, _ := json.Marshal(evaluation)
	iterMetrics := &evotypes.Metrics{
		TotalReturn: metrics.TotalReturnPct,
		MaxDrawdown: metrics.MaxDrawdownPct,
		WinRate:     metrics.WinRate,
		SharpeRatio: metrics.SharpeRatio,
		Trades:      metrics.Trades,
	}
	if err := e.store.Evolution().UpdateIterationComplete(
		e.evolutionID, version,
		iterMetrics,
		string(evalJSON),
		optimization.ExpectedEffect,
		optimization.NewPrompt,
	); err != nil {
		return fmt.Errorf("failed to update iteration: %w", err)
	}

	// 11. Update best version if improved
	// Improvement criteria:
	// 1. Higher return is always better
	// 2. Similar return (within 3%) but significantly better drawdown (5%+ improvement) is also considered improvement
	// Note: currentBestReturn, currentBestDrawdown, returnDiff, drawdownImprovement already calculated above

	isImproved := false
	improvementReason := ""

	if returnDiff > 0 {
		// Higher return - clear improvement
		isImproved = true
		improvementReason = fmt.Sprintf("higher return (%.2f%% vs %.2f%%)", metrics.TotalReturnPct, currentBestReturn)
	} else if returnDiff >= -3.0 && drawdownImprovement >= 5.0 {
		// Similar return (within 3%) but significantly better drawdown (5%+ improvement)
		isImproved = true
		improvementReason = fmt.Sprintf("similar return (%.2f%% vs %.2f%%) with better drawdown (%.2f%% vs %.2f%%)",
			metrics.TotalReturnPct, currentBestReturn, metrics.MaxDrawdownPct, currentBestDrawdown)
	}

	if isImproved {
		logger.Infof("Evolution %s: new best version %d - %s",
			e.evolutionID, version, improvementReason)
		e.updateBestVersion(version, metrics.TotalReturnPct, metrics.MaxDrawdownPct)
	} else {
		logger.Infof("Evolution %s v%d: no improvement (return %.2f%% vs best %.2f%%, drawdown %.2f%% vs best %.2f%%), will revert to best strategy",
			e.evolutionID, version, metrics.TotalReturnPct, currentBestReturn, metrics.MaxDrawdownPct, currentBestDrawdown)
	}

	// 12. Create or update strategy version with optimized prompt
	// Use evolution name as base, not current strategy name (to avoid name stacking like v3_v4_v5)
	baseStrategyName := e.config.Name
	if baseStrategyName == "" {
		// Fallback: extract base name from strategy (remove _vN suffix if present)
		baseStrategyName = strategy.Name
		if idx := strings.Index(baseStrategyName, "_v"); idx > 0 {
			baseStrategyName = baseStrategyName[:idx]
		}
	}
	strategyName := fmt.Sprintf("%s_v%d", baseStrategyName, version)
	existingStrategy, err := e.store.Strategy().GetByName(e.config.UserID, strategyName)

	var newStrategy *store.Strategy
	if err == nil && existingStrategy != nil {
		// Strategy with same name exists, update it
		existingStrategy.Config = optimization.NewPrompt
		existingStrategy.Description = fmt.Sprintf("Evolution iteration %d", version)
		if err := e.store.Strategy().Update(existingStrategy); err != nil {
			return fmt.Errorf("failed to update strategy: %w", err)
		}
		newStrategy = existingStrategy
		logger.Infof("Evolution %s: updated existing strategy %s", e.evolutionID, strategyName)
	} else {
		// Create new strategy
		newStrategy = &store.Strategy{
			ID:          uuid.New().String(),
			UserID:      e.config.UserID,
			Name:        strategyName,
			Config:      optimization.NewPrompt,
			Description: fmt.Sprintf("Evolution iteration %d", version),
		}
		if err := e.store.Strategy().Create(newStrategy); err != nil {
			return fmt.Errorf("failed to create new strategy: %w", err)
		}
		logger.Infof("Evolution %s: created new strategy %s", e.evolutionID, strategyName)
	}

	// 13. Update base_strategy_id for next iteration (both in memory and database)
	// Always use the AI-generated new strategy (optimization.NewPrompt) for next iteration
	// The AI has already analyzed current vs best and generated an optimized prompt
	e.config.BaseStrategyID = newStrategy.ID
	// Persist to database so resume works correctly
	if err := e.store.Evolution().UpdateBaseStrategy(e.evolutionID, newStrategy.ID); err != nil {
		logger.Warnf("Failed to persist base_strategy_id: %v", err)
	}
	logger.Infof("Evolution %s: using AI-optimized strategy %s for next iteration", e.evolutionID, newStrategy.ID)

	logger.Infof("Evolution %s v%d: iteration completed successfully", e.evolutionID, version)
	return nil
}

// waitForBacktestComplete waits for backtest to finish
func (e *AutoEvolver) waitForBacktestComplete(ctx context.Context, runID string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Activity-based timeout: if no progress update in 5 minutes, consider it stalled
	const inactivityTimeout = 5 * time.Minute
	lastProgress := float64(-1)
	lastActivityTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-e.stopChan:
			return fmt.Errorf("evolution stopped")
		case <-ticker.C:
			statusPayload := e.backtestMgr.Status(runID)

			// If runner is gone, check database for final state
			if statusPayload == nil {
				meta, err := e.backtestMgr.LoadMetadata(runID)
				if err != nil {
					return fmt.Errorf("failed to get backtest status: %w", err)
				}
				if meta == nil {
					return fmt.Errorf("backtest not found")
				}
				switch meta.State {
				case "completed":
					logger.Infof("Backtest %s completed (from metadata)", runID)
					return nil
				case "failed":
					return fmt.Errorf("backtest failed")
				default:
					// Runner gone but not completed - might be paused or stopped
					return fmt.Errorf("backtest ended with state: %s", meta.State)
				}
			}

			switch statusPayload.State {
			case "completed":
				return nil
			case "failed":
				return fmt.Errorf("backtest failed")
			case "running":
				// Check for activity: progress change or recent update
				currentProgress := statusPayload.ProgressPct

				// Check if progress has changed
				if currentProgress != lastProgress {
					lastProgress = currentProgress
					lastActivityTime = time.Now()
				} else {
					// Progress unchanged, check LastUpdatedIso
					if statusPayload.LastUpdatedIso != "" {
						if lastUpdate, err := time.Parse(time.RFC3339, statusPayload.LastUpdatedIso); err == nil {
							if time.Since(lastUpdate) < inactivityTimeout {
								// Recent update, reset activity time
								lastActivityTime = time.Now()
							}
						}
					}
				}

				// Check for inactivity timeout
				if time.Since(lastActivityTime) > inactivityTimeout {
					logger.Warnf("Backtest %s appears stalled: no progress update in %v (progress: %.1f%%)",
						runID, inactivityTimeout, currentProgress)
					return fmt.Errorf("backtest stalled: no activity for %v", inactivityTimeout)
				}
			default:
				logger.Warnf("Unknown backtest state: %s", statusPayload.State)
			}
		}
	}
}

// getIterationHistory returns summary of previous iterations for optimization context
func (e *AutoEvolver) getIterationHistory() []IterationSummary {
	iterations, err := e.store.Evolution().GetIterations(e.evolutionID)
	if err != nil {
		return nil
	}

	// Get best version info
	evolution, err := e.store.Evolution().Get(e.config.UserID, e.evolutionID)
	bestVersion := 0
	bestReturn := float64(-999999)
	bestDrawdown := float64(100)
	if err == nil {
		bestVersion = evolution.BestVersion
		bestReturn = evolution.BestReturn
		bestDrawdown = evolution.BestDrawdown
	}

	var history []IterationSummary
	for _, iter := range iterations {
		summary := IterationSummary{
			Version: iter.Version,
			Changes: iter.ChangesSummary,
		}
		if iter.Metrics != nil {
			summary.TotalReturn = iter.Metrics.TotalReturn
			summary.MaxDrawdown = iter.Metrics.MaxDrawdown
			// Mark as best if this is the best version
			summary.IsBest = iter.Version == bestVersion
			// Mark as failed using same criteria as improvement check:
			// Failed if: return is lower AND (return diff > 3% OR drawdown improvement < 5%)
			returnDiff := iter.Metrics.TotalReturn - bestReturn
			drawdownImprovement := bestDrawdown - iter.Metrics.MaxDrawdown
			isBetterReturn := returnDiff > 0
			isSimilarReturnWithBetterDrawdown := returnDiff >= -3.0 && drawdownImprovement >= 5.0
			summary.Failed = !summary.IsBest && !isBetterReturn && !isSimilarReturnWithBetterDrawdown
		}
		history = append(history, summary)
	}
	return history
}

// hydrateAIConfig fills in AI configuration from database
func (e *AutoEvolver) hydrateAIConfig(cfg *backtest.BacktestConfig) error {
	modelID := strings.TrimSpace(cfg.AIModelID)
	if modelID == "" {
		return fmt.Errorf("AI model ID is required")
	}

	model, err := e.store.AIModel().Get(cfg.UserID, modelID)
	if err != nil {
		return fmt.Errorf("failed to load AI model: %w", err)
	}

	if !model.Enabled {
		return fmt.Errorf("AI model %s is not enabled", model.Name)
	}

	apiKey := strings.TrimSpace(model.APIKey)
	if apiKey == "" {
		return fmt.Errorf("AI model %s is missing API Key", model.Name)
	}

	provider := strings.ToLower(strings.TrimSpace(model.Provider))
	if provider == "" || provider == "inherit" {
		modelNameLower := strings.ToLower(model.Name)
		if strings.Contains(modelNameLower, "claude") {
			provider = "anthropic"
		} else if strings.Contains(modelNameLower, "gpt") {
			provider = "openai"
		} else if strings.Contains(modelNameLower, "gemini") {
			provider = "google"
		} else if strings.Contains(modelNameLower, "deepseek") {
			provider = "deepseek"
		} else if model.CustomAPIURL != "" {
			provider = "custom"
		} else {
			provider = "openai"
		}
	}

	cfg.AICfg.Provider = provider
	cfg.AICfg.APIKey = apiKey
	cfg.AICfg.BaseURL = strings.TrimSpace(model.CustomAPIURL)
	cfg.AICfg.Model = strings.TrimSpace(model.CustomModelName)

	logger.Infof("Evolution AI config: provider=%s, model=%s", provider, cfg.AICfg.Model)
	return nil
}
