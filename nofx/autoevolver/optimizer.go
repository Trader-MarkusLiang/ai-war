package autoevolver

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"nofx/backtest"
	"nofx/evotypes"
	"nofx/logger"
	"nofx/mcp"
)

// Optimizer generates improved prompts based on evaluation
type Optimizer struct {
	aiClient mcp.AIClient
}

// NewOptimizer creates a new Optimizer
func NewOptimizer(aiClient mcp.AIClient) *Optimizer {
	return &Optimizer{aiClient: aiClient}
}

// OptimizationInput contains data needed for prompt optimization
type OptimizationInput struct {
	CurrentPrompt       string
	EvaluationReport    *evotypes.EvaluationReport
	IterationHistory    []IterationSummary
	// Best epoch data
	BestMetrics         *backtest.Metrics
	BestTrades          []backtest.TradeEvent
	BestVersion         int
	BestPrompt          string
	// Current epoch data (for comparison)
	CurrentMetrics      *backtest.Metrics
	CurrentTrades       []backtest.TradeEvent
	CurrentVersion      int
	IsCurrentBest       bool // true if current epoch is the best
}

// IterationSummary contains key info from previous iterations
type IterationSummary struct {
	Version     int     `json:"version"`
	TotalReturn float64 `json:"total_return"`
	MaxDrawdown float64 `json:"max_drawdown"`
	Changes     string  `json:"changes"`
	IsBest      bool    `json:"is_best"`
	Failed      bool    `json:"failed"` // true if this iteration performed worse than best
}

// Optimize generates an improved prompt based on evaluation
func (o *Optimizer) Optimize(input *OptimizationInput) (*evotypes.OptimizationResult, error) {
	if o.aiClient == nil {
		return o.createFallbackResult(input), nil
	}

	systemPrompt := buildOptimizationSystemPrompt()
	userPrompt := buildOptimizationUserPrompt(input)

	logger.Infof("Optimizer: calling AI for prompt optimization...")

	response, err := o.aiClient.CallWithMessages(systemPrompt, userPrompt)
	if err != nil {
		logger.Warnf("AI optimization failed, using fallback: %v", err)
		return o.createFallbackResult(input), nil
	}

	result, err := parseOptimizationResponse(response, input.CurrentPrompt)
	if err != nil {
		logger.Warnf("Failed to parse optimization response: %v", err)
		return o.createFallbackResult(input), nil
	}
	result.RawResponse = response

	return result, nil
}

// createFallbackResult returns original prompt when AI is unavailable
func (o *Optimizer) createFallbackResult(input *OptimizationInput) *evotypes.OptimizationResult {
	return &evotypes.OptimizationResult{
		Changes:        []string{"No AI optimization available - keeping original prompt"},
		NewPrompt:      input.CurrentPrompt,
		ExpectedEffect: "No changes applied",
	}
}

func buildOptimizationSystemPrompt() string {
	return `You are an expert trading strategy prompt engineer for crypto futures. Your task is to improve a Stoch RSI + EMA + MACD strategy.

## CRITICAL RULES - READ CAREFULLY

### 1. Core Trading Principle: Position Size Matters More Than Win Rate
- **Single large winning trades often determine total profitability**
- A 50% win rate with proper position sizing beats 60% win rate with small positions
- DO NOT sacrifice position size for marginal improvements in win rate
- Example: 1.6 ETH position earning +435 USDT > 0.2 ETH position earning +65 USDT (same price move)

### 2. FORBIDDEN Changes (DO NOT DO THESE)
- DO NOT reduce max_margin_usage below 0.5
- DO NOT reduce position sizes below 30% for any confidence level
- DO NOT reduce base position size below 50% of account when 3 conditions met
- DO NOT add more than 2 mandatory conditions for entry
- DO NOT reduce trading frequency by more than 30%
- DO NOT add ATR-based stops if previous attempts with ATR failed
- **CRITICAL: DO NOT make strategies more conservative just because you see drawdown**

### 3. Parameter Boundaries (STRICT)
- max_margin_usage: 0.6 - 0.9 (prefer 0.7-0.9)
- min_confidence: 65 - 80
- position_size (base, 1 condition): 30% - 40%
- position_size (2 conditions): 50% - 70%
- position_size (3 conditions): 70% - 100%
- max_positions: 2 - 4
- leverage: 4 - 5x (prefer 5x)

### 4. Optimization Strategy Priority

**Priority 1: Fix stop-loss execution issues**
- If you see losses exceeding -2% (e.g., -4.8%, -3.8%), the stop-loss is NOT being executed properly
- Add specific numerical examples: "开仓价 90000, 止损价 = 90000×0.98 = 88200"
- Add explicit check steps: "1. Calculate stop price 2. Compare current price 3. If triggered → close immediately"
- Consider tiered stop-loss: -1.5% warning, -2% mandatory, -3% final defense
- Make stop-loss the FIRST priority in decision flow, before any other conditions

**Priority 2: Prevent chasing highs/lows**
- If you see pattern: "short profit → immediately long at higher price → loss"
- Add reverse trade cooldown period (at least 1 period/4 hours)
- Example: "After closing short, wait 1 period before opening long"

**Priority 3: Maintain or increase position sizes**
- Large positions on high-confidence signals = key to profitability
- If current version has small positions, INCREASE them
- DO NOT reduce positions just because you see drawdown

**Priority 4: Improve entry timing**
- Adjust Stoch RSI thresholds (e.g., ≤25 for long, ≥75 for short)
- Fine-tune EMA/MACD confirmation logic
- DO NOT add new mandatory conditions

**Priority 5: Optimize exit management**
- Let profits run (don't exit too early on winning trades)
- Quick stop-loss on losing trades
- Improve trailing stop logic

**Priority 6 (LAST RESORT): Add filters**
- Only if 3+ iterations show consistent pattern of bad entries
- Must not reduce position sizes

### 5. Data-Driven Optimization Framework

**CRITICAL: Analyze specific trades, not just overall metrics**

When you see poor performance, identify the specific problem pattern:

1. **Stop-loss execution failure**: Losses exceeding -2% (e.g., -4.8%, -3.8%)
   - Solution: Add numerical examples, explicit check steps, tiered stop-loss system

2. **Chasing highs/lows**: Pattern of "profit → reverse at worse price → loss"
   - Solution: Add reverse trade cooldown period (1-2 periods)

3. **Position sizing too small**: Winning trades can't offset losses
   - Solution: Increase position sizes, maintain aggressive sizing

4. **Poor entry timing**: Entries at extreme prices without confirmation
   - Solution: Tighten thresholds, add price position checks

5. **Premature exits**: Profits exited too early, losses held too long
   - Solution: Let profits run, quick stop-loss on losses

**DO NOT make generic changes without identifying specific problem patterns.**

### 6. Learning from Failures
When you see failed iterations:
- If they reduced position sizes → REVERT and increase positions
- If they added filters → REMOVE filters and adjust existing parameters
- If 3+ consecutive failures → GO BACK to best version's position sizing
- **Remember: Missing big moves due to small positions is worse than occasional losses**

## Response Format (JSON)
{
  "changes": ["one specific change"],
  "new_prompt": "complete strategy prompt JSON",
  "expected_effect": "expected improvement",
  "reasoning": "why this change should work"
}

The new_prompt must be valid JSON that can be parsed directly.`
}

func buildOptimizationUserPrompt(input *OptimizationInput) string {
	var sb strings.Builder

	sb.WriteString("## Current Strategy Prompt\n\n")
	sb.WriteString("```\n")
	sb.WriteString(input.CurrentPrompt)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Evaluation Results\n\n")
	if input.EvaluationReport != nil {
		sb.WriteString("### Strengths\n")
		for _, s := range input.EvaluationReport.Strengths {
			sb.WriteString(fmt.Sprintf("- %s\n", s))
		}

		sb.WriteString("\n### Weaknesses\n")
		for _, w := range input.EvaluationReport.Weaknesses {
			sb.WriteString(fmt.Sprintf("- %s\n", w))
		}

		sb.WriteString("\n### Suggestions\n")
		for _, s := range input.EvaluationReport.Suggestions {
			sb.WriteString(fmt.Sprintf("- %s\n", s))
		}
	}

	if len(input.IterationHistory) > 0 {
		sb.WriteString("\n## Previous Iterations Analysis\n\n")

		// Find and highlight the best iteration
		var bestIter *IterationSummary
		var failedIters []IterationSummary
		consecutiveFailures := 0
		for i := range input.IterationHistory {
			iter := &input.IterationHistory[i]
			if iter.IsBest {
				bestIter = iter
				consecutiveFailures = 0
			}
			if iter.Failed {
				failedIters = append(failedIters, *iter)
				consecutiveFailures++
			}
		}

		// Show best iteration prominently
		if bestIter != nil {
			sb.WriteString(fmt.Sprintf("### 🏆 Best Performing (v%d): Return %.2f%%, Drawdown %.2f%%\n",
				bestIter.Version, bestIter.TotalReturn, bestIter.MaxDrawdown))
			sb.WriteString("**This is the baseline. Your goal is to improve upon this, not make it worse.**\n\n")
		}

		// Warn about consecutive failures
		if consecutiveFailures >= 3 {
			sb.WriteString(fmt.Sprintf("### ⚠️ WARNING: %d consecutive failed iterations!\n", consecutiveFailures))
			sb.WriteString("**STOP adding complexity. Consider SIMPLIFYING the strategy instead.**\n\n")
		}

		// Analyze failure patterns
		if len(failedIters) > 0 {
			sb.WriteString("### ❌ Failed Attempts - Learn from these mistakes:\n")
			for _, iter := range failedIters {
				sb.WriteString(fmt.Sprintf("- v%d: Return %.2f%% - %s\n",
					iter.Version, iter.TotalReturn, iter.Changes))
			}
			sb.WriteString("\n**Pattern Analysis**: Look at what these failed attempts have in common. DO NOT repeat similar changes.\n\n")
		}

		// Show all iterations for context
		sb.WriteString("### All Iterations:\n")
		for _, iter := range input.IterationHistory {
			status := ""
			if iter.IsBest {
				status = " ✅ [BEST]"
			} else if iter.Failed {
				status = " ❌ [FAILED]"
			}
			sb.WriteString(fmt.Sprintf("- v%d: Return %.2f%%, Drawdown %.2f%%%s\n",
				iter.Version, iter.TotalReturn, iter.MaxDrawdown, status))
		}
	}

	// Comparison Analysis: Current vs Best epoch
	if !input.IsCurrentBest && input.BestVersion > 0 {
		sb.WriteString("\n## Performance Comparison: Current vs Best\n\n")
		sb.WriteString(fmt.Sprintf("**Current Epoch (v%d)** vs **Best Epoch (v%d)**\n\n", input.CurrentVersion, input.BestVersion))

		// Metrics comparison
		if input.CurrentMetrics != nil && input.BestMetrics != nil {
			sb.WriteString("### Metrics Comparison\n")
			sb.WriteString("| Metric | Current | Best | Diff |\n")
			sb.WriteString("|--------|---------|------|------|\n")
			sb.WriteString(fmt.Sprintf("| Return | %.2f%% | %.2f%% | %.2f%% |\n",
				input.CurrentMetrics.TotalReturnPct, input.BestMetrics.TotalReturnPct,
				input.CurrentMetrics.TotalReturnPct-input.BestMetrics.TotalReturnPct))
			sb.WriteString(fmt.Sprintf("| Drawdown | %.2f%% | %.2f%% | %.2f%% |\n",
				input.CurrentMetrics.MaxDrawdownPct, input.BestMetrics.MaxDrawdownPct,
				input.CurrentMetrics.MaxDrawdownPct-input.BestMetrics.MaxDrawdownPct))
			sb.WriteString(fmt.Sprintf("| Win Rate | %.1f%% | %.1f%% | %.1f%% |\n",
				input.CurrentMetrics.WinRate*100, input.BestMetrics.WinRate*100,
				(input.CurrentMetrics.WinRate-input.BestMetrics.WinRate)*100))
			sb.WriteString(fmt.Sprintf("| Trades | %d | %d | %d |\n\n",
				input.CurrentMetrics.Trades, input.BestMetrics.Trades,
				input.CurrentMetrics.Trades-input.BestMetrics.Trades))
		}

		// Current epoch trades
		if len(input.CurrentTrades) > 0 {
			sb.WriteString(fmt.Sprintf("### Current Epoch (v%d) Sample Trades\n", input.CurrentVersion))
			writeTradesSummary(&sb, input.CurrentTrades, 5)
		}

		// Best epoch trades
		if len(input.BestTrades) > 0 {
			sb.WriteString(fmt.Sprintf("### Best Epoch (v%d) Sample Trades\n", input.BestVersion))
			writeTradesSummary(&sb, input.BestTrades, 5)
		}

		sb.WriteString("\n**Analysis Task**: Compare the trading decisions between current and best epoch. ")
		sb.WriteString("Identify what the best epoch did differently that led to better performance. ")
		sb.WriteString("The optimization should be based on the BEST epoch's strategy.\n")
	} else {
		// Current is best, just show its trades
		if len(input.CurrentTrades) > 0 {
			sb.WriteString("\n## Current Best Epoch Trade Analysis\n\n")
			writeTradesSummary(&sb, input.CurrentTrades, 5)
		}
		if input.CurrentMetrics != nil {
			sb.WriteString("\n## Current Metrics\n\n")
			m := input.CurrentMetrics
			sb.WriteString(fmt.Sprintf("- Total Return: %.2f%%\n", m.TotalReturnPct))
			sb.WriteString(fmt.Sprintf("- Max Drawdown: %.2f%%\n", m.MaxDrawdownPct))
			sb.WriteString(fmt.Sprintf("- Win Rate: %.2f%%\n", m.WinRate*100))
			sb.WriteString(fmt.Sprintf("- Sharpe Ratio: %.2f\n", m.SharpeRatio))
			sb.WriteString(fmt.Sprintf("- Total Trades: %d\n", m.Trades))
		}
	}

	sb.WriteString("\nPlease optimize the prompt and respond in JSON format.")

	return sb.String()
}

// writeTradesSummary writes a summary of trades to the string builder
func writeTradesSummary(sb *strings.Builder, trades []backtest.TradeEvent, limit int) {
	if len(trades) == 0 {
		sb.WriteString("No trades recorded.\n\n")
		return
	}

	// Show up to limit trades
	count := len(trades)
	if count > limit {
		count = limit
	}

	sb.WriteString("| Time | Action | Price | Note |\n")
	sb.WriteString("|------|--------|-------|------|\n")

	for i := 0; i < count; i++ {
		t := trades[i]
		note := t.Note
		if len(note) > 50 {
			note = note[:47] + "..."
		}
		// Convert timestamp (milliseconds) to time
		tradeTime := time.Unix(t.Timestamp/1000, 0)
		sb.WriteString(fmt.Sprintf("| %s | %s | %.2f | %s |\n",
			tradeTime.Format("01-02 15:04"), t.Action, t.Price, note))
	}

	if len(trades) > limit {
		sb.WriteString(fmt.Sprintf("\n... and %d more trades\n", len(trades)-limit))
	}
	sb.WriteString("\n")
}

func parseOptimizationResponse(response, currentPrompt string) (*evotypes.OptimizationResult, error) {
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var result evotypes.OptimizationResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate new prompt is not empty
	if strings.TrimSpace(result.NewPrompt) == "" {
		result.NewPrompt = currentPrompt
		result.Changes = append(result.Changes, "Warning: AI returned empty prompt, keeping original")
	}

	return &result, nil
}
