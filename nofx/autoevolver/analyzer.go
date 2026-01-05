package autoevolver

import (
	"encoding/json"
	"fmt"
	"strings"

	"nofx/backtest"
	"nofx/evotypes"
	"nofx/logger"
	"nofx/mcp"
)

// Analyzer evaluates backtest results using AI
type Analyzer struct {
	aiClient mcp.AIClient
}

// NewAnalyzer creates a new Analyzer
func NewAnalyzer(aiClient mcp.AIClient) *Analyzer {
	return &Analyzer{aiClient: aiClient}
}

// AnalysisInput contains all data needed for AI evaluation
type AnalysisInput struct {
	Metrics       *backtest.Metrics
	CurrentPrompt string
	Trades        []backtest.TradeEvent
	Decisions     []DecisionSample
	EquityCurve   []backtest.EquityPoint
}

// Analyze evaluates backtest results and returns an evaluation report
func (a *Analyzer) Analyze(input *AnalysisInput) (*evotypes.EvaluationReport, error) {
	if a.aiClient == nil {
		return a.createFallbackReport(input.Metrics), nil
	}

	// Build analysis prompt
	systemPrompt := buildAnalysisSystemPrompt()
	userPrompt := buildAnalysisUserPrompt(input)

	logger.Infof("Analyzer: calling AI for evaluation...")

	// Call AI
	response, err := a.aiClient.CallWithMessages(systemPrompt, userPrompt)
	if err != nil {
		logger.Warnf("AI analysis failed, using fallback: %v", err)
		return a.createFallbackReport(input.Metrics), nil
	}

	// Parse response
	report, err := parseEvaluationResponse(response)
	if err != nil {
		logger.Warnf("Failed to parse AI response, using fallback: %v", err)
		report = a.createFallbackReport(input.Metrics)
	}
	report.RawResponse = response

	return report, nil
}

// createFallbackReport creates a basic report when AI is unavailable
func (a *Analyzer) createFallbackReport(metrics *backtest.Metrics) *evotypes.EvaluationReport {
	report := &evotypes.EvaluationReport{
		Strengths:   []string{},
		Weaknesses:  []string{},
		Suggestions: []string{},
	}

	if metrics == nil {
		report.Weaknesses = append(report.Weaknesses, "No metrics available")
		return report
	}

	// Analyze metrics
	if metrics.TotalReturnPct > 0 {
		report.Strengths = append(report.Strengths,
			fmt.Sprintf("Positive return: %.2f%%", metrics.TotalReturnPct))
	} else {
		report.Weaknesses = append(report.Weaknesses,
			fmt.Sprintf("Negative return: %.2f%%", metrics.TotalReturnPct))
	}

	if metrics.MaxDrawdownPct < 20 {
		report.Strengths = append(report.Strengths,
			fmt.Sprintf("Controlled drawdown: %.2f%%", metrics.MaxDrawdownPct))
	} else {
		report.Weaknesses = append(report.Weaknesses,
			fmt.Sprintf("High drawdown: %.2f%%", metrics.MaxDrawdownPct))
	}

	if metrics.WinRate > 50 {
		report.Strengths = append(report.Strengths,
			fmt.Sprintf("Good win rate: %.1f%%", metrics.WinRate))
	} else {
		report.Weaknesses = append(report.Weaknesses,
			fmt.Sprintf("Low win rate: %.1f%%", metrics.WinRate))
	}

	if metrics.SharpeRatio > 1 {
		report.Strengths = append(report.Strengths,
			fmt.Sprintf("Good Sharpe ratio: %.2f", metrics.SharpeRatio))
	} else if metrics.SharpeRatio < 0 {
		report.Weaknesses = append(report.Weaknesses,
			fmt.Sprintf("Negative Sharpe ratio: %.2f", metrics.SharpeRatio))
	}

	// Generate suggestions based on weaknesses
	if metrics.MaxDrawdownPct > 20 {
		report.Suggestions = append(report.Suggestions,
			"Consider tighter stop-loss rules to reduce drawdown")
	}
	if metrics.WinRate < 50 {
		report.Suggestions = append(report.Suggestions,
			"Improve entry signal accuracy or add confirmation filters")
	}
	if metrics.TotalReturnPct < 0 {
		report.Suggestions = append(report.Suggestions,
			"Review position sizing and risk management rules")
	}

	return report
}

func buildAnalysisSystemPrompt() string {
	return `You are an expert quantitative trading strategy analyst specializing in crypto futures trading with Stoch RSI + EMA + MACD strategies.

## Your Task
Analyze backtest results and provide actionable insights. Focus on WHAT WORKED and WHAT DIDN'T.

## Analysis Framework

### 1. Trade Quality Analysis
- Compare winning trades vs losing trades patterns
- Identify which market conditions (trending/ranging) the strategy performs best
- Analyze if long/short trades have different success rates

### 2. Signal Quality
- Are entry signals too early/late?
- Are exit signals leaving money on the table or holding too long?
- Is the strategy over-filtering (missing opportunities) or under-filtering (too many bad trades)?

### 3. Risk Management
- Is position sizing appropriate?
- Are stop losses too tight (stopped out before reversal) or too loose (large losses)?
- Is the strategy taking enough trades to be statistically meaningful (target: 30-50 trades/month)?

## IMPORTANT PRINCIPLES
- Simple strategies often outperform complex ones
- More filters ≠ better performance (often reduces opportunities)
- Trading frequency matters: too few trades = not enough data, too many = overtrading
- A 40% win rate with 2:1 reward/risk is profitable

## Response Format (JSON)
{
  "strengths": ["specific strength with data"],
  "weaknesses": ["specific weakness with data"],
  "suggestions": ["actionable suggestion - be specific about what to change"],
  "trade_pattern": "brief description of winning vs losing trade patterns"
}

Keep each point concise. Provide 2-3 items per category.`
}

func buildAnalysisUserPrompt(input *AnalysisInput) string {
	var sb strings.Builder

	sb.WriteString("## Backtest Results\n\n")

	if input.Metrics != nil {
		sb.WriteString(fmt.Sprintf("- Total Return: %.2f%%\n", input.Metrics.TotalReturnPct))
		sb.WriteString(fmt.Sprintf("- Max Drawdown: %.2f%%\n", input.Metrics.MaxDrawdownPct))
		sb.WriteString(fmt.Sprintf("- Win Rate: %.1f%%\n", input.Metrics.WinRate))
		sb.WriteString(fmt.Sprintf("- Sharpe Ratio: %.2f\n", input.Metrics.SharpeRatio))
		sb.WriteString(fmt.Sprintf("- Total Trades: %d\n", input.Metrics.Trades))
		sb.WriteString(fmt.Sprintf("- Profit Factor: %.2f\n", input.Metrics.ProfitFactor))
	}

	// Add detailed trade analysis
	if len(input.Trades) > 0 {
		sb.WriteString("\n## Trade Analysis\n\n")

		// Separate winning and losing trades
		var winningTrades, losingTrades []backtest.TradeEvent
		var longTrades, shortTrades int
		var longWins, shortWins int
		var totalProfit, totalLoss float64

		for _, t := range input.Trades {
			if t.RealizedPnL > 0 {
				winningTrades = append(winningTrades, t)
				totalProfit += t.RealizedPnL
			} else if t.RealizedPnL < 0 {
				losingTrades = append(losingTrades, t)
				totalLoss += t.RealizedPnL
			}

			if t.Side == "long" || t.Action == "open_long" || t.Action == "close_long" {
				longTrades++
				if t.RealizedPnL > 0 {
					longWins++
				}
			} else if t.Side == "short" || t.Action == "open_short" || t.Action == "close_short" {
				shortTrades++
				if t.RealizedPnL > 0 {
					shortWins++
				}
			}
		}

		sb.WriteString(fmt.Sprintf("### Summary\n"))
		sb.WriteString(fmt.Sprintf("- Winning Trades: %d (avg profit: %.2f USDT)\n",
			len(winningTrades), safeAvg(totalProfit, len(winningTrades))))
		sb.WriteString(fmt.Sprintf("- Losing Trades: %d (avg loss: %.2f USDT)\n",
			len(losingTrades), safeAvg(totalLoss, len(losingTrades))))

		if longTrades > 0 {
			sb.WriteString(fmt.Sprintf("- Long Trades: %d (win rate: %.1f%%)\n",
				longTrades, float64(longWins)/float64(longTrades)*100))
		}
		if shortTrades > 0 {
			sb.WriteString(fmt.Sprintf("- Short Trades: %d (win rate: %.1f%%)\n",
				shortTrades, float64(shortWins)/float64(shortTrades)*100))
		}

		// Show sample winning trades
		if len(winningTrades) > 0 {
			sb.WriteString("\n### Sample Winning Trades\n")
			count := min(3, len(winningTrades))
			for i := 0; i < count; i++ {
				t := winningTrades[i]
				sb.WriteString(fmt.Sprintf("- %s %s: +%.2f USDT\n", t.Symbol, t.Action, t.RealizedPnL))
			}
		}

		// Show sample losing trades
		if len(losingTrades) > 0 {
			sb.WriteString("\n### Sample Losing Trades\n")
			count := min(3, len(losingTrades))
			for i := 0; i < count; i++ {
				t := losingTrades[i]
				sb.WriteString(fmt.Sprintf("- %s %s: %.2f USDT\n", t.Symbol, t.Action, t.RealizedPnL))
			}
		}
	}

	sb.WriteString("\n## Current Strategy Prompt (truncated)\n\n")
	sb.WriteString("```\n")
	sb.WriteString(truncateString(input.CurrentPrompt, 1500))
	sb.WriteString("\n```\n")

	sb.WriteString("\nPlease analyze these results and provide your evaluation in JSON format.")

	return sb.String()
}

func safeAvg(total float64, count int) float64 {
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseEvaluationResponse(response string) (*evotypes.EvaluationReport, error) {
	// Try to extract JSON from response
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var report evotypes.EvaluationReport
	if err := json.Unmarshal([]byte(jsonStr), &report); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &report, nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func extractJSON(s string) string {
	// Find JSON object in response
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}

	// Find matching closing brace
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}

	return ""
}
