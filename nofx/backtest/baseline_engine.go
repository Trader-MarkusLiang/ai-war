package backtest

import (
	"nofx/decision"
	"nofx/logger"
	"nofx/market"
	"nofx/store"
)

// BaselineEngine 传统指标决策引擎（确定性）
// 基于技术指标生成确定性的交易决策，作为 AI 决策的基线对比
type BaselineEngine struct {
	config         *store.StrategyConfig
	positionStates map[string]*BaselinePositionState // 持仓状态跟踪
}

// BaselinePositionState 持仓状态跟踪（用于移动止盈止损）
type BaselinePositionState struct {
	Symbol        string
	Side          string
	EntryPrice    float64
	PeakPrice     float64 // 多头最高价/空头最低价
	TrailingStop  float64 // 当前移动止损位
	TrailingTP    float64 // 当前移动止盈位
	HardStopPrice float64 // 挂单硬止损价（开仓时设置，基于OHLC检查）
	EntryCycle    int     // 开仓时的周期数（用于最小持仓周期检查）
}

// ScoredDecision 带评分的决策（用于筛选最优开仓决策）
type ScoredDecision struct {
	Decision decision.Decision
	Score    float64 // 综合评分（0-100）
}

// NewBaselineEngine 创建传统指标引擎
func NewBaselineEngine(config *store.StrategyConfig) *BaselineEngine {
	return &BaselineEngine{
		config:         config,
		positionStates: make(map[string]*BaselinePositionState),
	}
}

// MakeDecision 基于技术指标生成确定性决策
// 输入相同的市场数据，输出相同的决策（确定性）
func (e *BaselineEngine) MakeDecision(
	equity float64,
	available float64,
	marketData map[string]*market.Data,
	positions []decision.PositionInfo,
) []decision.Decision {
	finalDecisions := make([]decision.Decision, 0)

	// 1. 更新持仓状态（峰值价格）
	for _, pos := range positions {
		if data, ok := marketData[pos.Symbol]; ok {
			e.updatePositionState(pos, data.CurrentPrice)
		}
	}

	// 2. 优先处理平仓决策（最高优先级）
	closeDecisions := make([]decision.Decision, 0)
	for _, pos := range positions {
		if data, ok := marketData[pos.Symbol]; ok {
			if closeDecision := e.checkExitSignal(pos, data); closeDecision != nil {
				closeDecisions = append(closeDecisions, *closeDecision)
				// 清除持仓状态
				delete(e.positionStates, pos.Symbol+"_"+pos.Side)
			}
		}
	}
	finalDecisions = append(finalDecisions, closeDecisions...)

	// 3. 生成所有候选开仓决策（不限制数量）
	if available > 100 { // 至少 100 USDT 才考虑开仓
		candidateDecisions := make([]ScoredDecision, 0)

		for symbol, data := range marketData {
			if !e.hasPosition(positions, symbol) {
				// 生成候选决策并计算评分
				if scoredDec := e.generateScoredDecision(symbol, data, equity, available); scoredDec != nil {
					candidateDecisions = append(candidateDecisions, *scoredDec)
				}
			}
		}

		// 4. 根据评分筛选最优的开仓决策
		selectedDecisions := e.selectBestDecisions(candidateDecisions, len(positions))
		finalDecisions = append(finalDecisions, selectedDecisions...)
	}

	return finalDecisions
}

// checkEntrySignal 检查入场信号
func (e *BaselineEngine) checkEntrySignal(
	symbol string,
	data *market.Data,
	equity float64,
	available float64,
	currentPositions int,
) *decision.Decision {
	if data == nil {
		return nil
	}

	// 检查是否超过最大持仓数
	maxPositions := e.config.RiskControl.MaxPositions
	if maxPositions <= 0 {
		maxPositions = 3
	}
	if currentPositions >= maxPositions {
		return nil
	}

	// 获取指标配置
	indicators := e.config.Indicators

	// 获取当前指标值
	price := data.CurrentPrice
	rsi7 := data.CurrentRSI7
	macd := data.CurrentMACD
	ema20 := data.CurrentEMA20

	// 信号计数
	longSignals := 0
	shortSignals := 0

	// RSI 信号
	if indicators.EnableRSI && rsi7 > 0 {
		if rsi7 < 25 {
			longSignals++ // 超卖 -> 做多信号（优化：从 30 收紧到 25）
		} else if rsi7 > 75 {
			shortSignals++ // 超买 -> 做空信号（优化：从 70 收紧到 75）
		}
	}

	// MACD 信号
	if indicators.EnableMACD {
		if macd > 0 {
			longSignals++
		} else if macd < 0 {
			shortSignals++
		}
	}

	// EMA 趋势信号
	if indicators.EnableEMA && ema20 > 0 {
		if price > ema20 {
			longSignals++ // 价格在 EMA 上方 -> 上升趋势
		} else {
			shortSignals++ // 价格在 EMA 下方 -> 下降趋势
		}
	}

	// Stoch RSI 信号
	if indicators.EnableStochRSI {
		k, d := e.getStochRSI(data)
		if k > 0 && d > 0 {
			if k < 15 && k > d { // 超卖区金叉（优化：从 20 收紧到 15）
				longSignals++
			} else if k > 85 && k < d { // 超买区死叉（优化：从 80 收紧到 85）
				shortSignals++
			}
		}
	}

	// 获取配置参数
	baselineCfg := e.config.BaselineConfig
	if baselineCfg == nil {
		return nil // 没有 Baseline 配置
	}

	// 需要至少 N 个信号共振才开仓
	minSignals := baselineCfg.SignalThresholds.MinSignalCount
	if minSignals <= 0 {
		minSignals = 3 // 默认值（优化：从 2 提高到 3）
	}

	// 计算仓位大小（基于可用资金和最大持仓数）
	leverage := baselineCfg.RiskManagement.Leverage
	if leverage <= 0 {
		leverage = 5 // 默认值
	}

	// 计算单个仓位大小 = 可用资金 / 最大持仓数
	// 这样可以确保有足够的资金开仓
	// 注意: maxPositions 已经在函数开头获取过了
	maxPos := e.config.RiskControl.MaxPositions
	if maxPos <= 0 {
		maxPos = 3
	}
	positionValue := (available / float64(maxPos)) * float64(leverage)
	if positionValue < 50 {
		return nil // 仓位太小
	}

	// 硬止损百分比
	hardStopLossPct := baselineCfg.RiskManagement.HardStopLossPct
	if hardStopLossPct <= 0 {
		hardStopLossPct = 3.0 // 默认 -3.0%
	}

	// 注意：总仓位限制（本金的 90%）在回测环境中由 runner 层面处理

	if longSignals >= minSignals {
		// 🔧 BUG FIX: 检查是否已有相同方向的持仓状态
		// 防止状态覆盖导致止损失效
		stateKey := symbol + "_long"
		if _, exists := e.positionStates[stateKey]; exists {
			// 已有多头持仓，拒绝开新仓
			return nil
		}

		stopLossPrice := price * (1 - hardStopLossPct/100)
		// 初始化持仓状态
		e.positionStates[stateKey] = &BaselinePositionState{
			Symbol:       symbol,
			Side:         "long",
			EntryPrice:   price,
			PeakPrice:    price,
			TrailingStop: stopLossPrice,
			TrailingTP:   0, // 初始无移动止盈
		}

		return &decision.Decision{
			Symbol:          symbol,
			Action:          "open_long",
			Leverage:        leverage,
			PositionSizeUSD: positionValue,
			StopLoss:        stopLossPrice,
			TakeProfit:      0, // 不设固定止盈,使用移动止盈
			Confidence:      75,
			Reasoning:       "Baseline: Multiple long signals (RSI/MACD/EMA/StochRSI)",
		}
	}

	if shortSignals >= minSignals {
		// 🔧 BUG FIX: 检查是否已有相同方向的持仓状态
		// 防止状态覆盖导致止损失效
		stateKey := symbol + "_short"
		if _, exists := e.positionStates[stateKey]; exists {
			// 已有空头持仓，拒绝开新仓
			return nil
		}

		stopLossPrice := price * (1 + hardStopLossPct/100)
		// 初始化持仓状态
		e.positionStates[stateKey] = &BaselinePositionState{
			Symbol:       symbol,
			Side:         "short",
			EntryPrice:   price,
			PeakPrice:    price,
			TrailingStop: stopLossPrice,
			TrailingTP:   0, // 初始无移动止盈
		}

		return &decision.Decision{
			Symbol:          symbol,
			Action:          "open_short",
			Leverage:        leverage,
			PositionSizeUSD: positionValue,
			StopLoss:        stopLossPrice,
			TakeProfit:      0, // 不设固定止盈,使用移动止盈
			Confidence:      75,
			Reasoning:       "Baseline: Multiple short signals (RSI/MACD/EMA/StochRSI)",
		}
	}

	return nil
}

// checkExitSignal 检查出场信号（按优先级执行）
func (e *BaselineEngine) checkExitSignal(pos decision.PositionInfo, data *market.Data) *decision.Decision {
	if data == nil {
		return nil
	}

	currentPrice := data.CurrentPrice
	pnlPct := pos.UnrealizedPnLPct

	// 获取配置参数
	baselineCfg := e.config.BaselineConfig
	if baselineCfg == nil {
		return nil // 没有 Baseline 配置
	}
	hardStopLossPct := baselineCfg.RiskManagement.HardStopLossPct
	if hardStopLossPct <= 0 {
		hardStopLossPct = 2.5 // 默认 -2.5%（优化版）
	}

	// 获取持仓状态
	stateKey := pos.Symbol + "_" + pos.Side
	state, hasState := e.positionStates[stateKey]
	if !hasState {
		// 如果没有状态记录,创建一个（兼容旧数据）
		state = &BaselinePositionState{
			Symbol:     pos.Symbol,
			Side:       pos.Side,
			EntryPrice: pos.EntryPrice,
			PeakPrice:  currentPrice,
		}
		if pos.Side == "long" {
			state.TrailingStop = pos.EntryPrice * (1 - hardStopLossPct/100)
		} else {
			state.TrailingStop = pos.EntryPrice * (1 + hardStopLossPct/100)
		}
		e.positionStates[stateKey] = state
	}

	action := "close_long"
	if pos.Side == "short" {
		action = "close_short"
	}

	if pos.Side == "long" {
		return e.checkLongExit(pos, currentPrice, pnlPct, state, action, data, baselineCfg)
	} else {
		return e.checkShortExit(pos, currentPrice, pnlPct, state, action, data, baselineCfg)
	}
}

// checkLongExit 检查多头出场信号（按优先级）
func (e *BaselineEngine) checkLongExit(
	pos decision.PositionInfo,
	currentPrice float64,
	pnlPct float64,
	state *BaselinePositionState,
	action string,
	data *market.Data,
	cfg *store.BaselineConfig,
) *decision.Decision {
	// 1. 强制止损（CRITICAL - 最高优先级）
	hardStopLossPct := cfg.RiskManagement.HardStopLossPct
	if hardStopLossPct <= 0 {
		hardStopLossPct = 3.0 // 默认 -3.0%
	}
	if currentPrice <= state.EntryPrice*(1-hardStopLossPct/100) {
		return &decision.Decision{
			Symbol:    pos.Symbol,
			Action:    action,
			Reasoning: "Baseline: Hard stop loss (CRITICAL)",
		}
	}

	// 2. 终极止盈：RSI > 70 或 StochRSI 死叉
	rsi7 := data.CurrentRSI7
	if e.config.Indicators.EnableRSI && rsi7 > 70 {
		return &decision.Decision{
			Symbol:    pos.Symbol,
			Action:    action,
			Reasoning: "Baseline: RSI overbought exit (>70)",
		}
	}

	k, d := e.getStochRSI(data)
	// StochRSI 出场：要求 K 值在超买区（>=70）且死叉
	// 优化：提高触发门槛，减少频繁出场
	if e.config.Indicators.EnableStochRSI && k >= 70 && k < d {
		return &decision.Decision{
			Symbol:    pos.Symbol,
			Action:    action,
			Reasoning: "Baseline: StochRSI death cross exit",
		}
	}

	// 3. 移动止盈
	if state.TrailingTP > 0 && currentPrice <= state.TrailingTP {
		return &decision.Decision{
			Symbol:    pos.Symbol,
			Action:    action,
			Reasoning: "Baseline: Trailing take profit triggered",
		}
	}

	// 4. 移动止损
	if pnlPct >= 3.0 && currentPrice <= state.TrailingStop {
		return &decision.Decision{
			Symbol:    pos.Symbol,
			Action:    action,
			Reasoning: "Baseline: Trailing stop loss triggered",
		}
	}

	return nil
}

// checkShortExit 检查空头出场信号（按优先级）
func (e *BaselineEngine) checkShortExit(
	pos decision.PositionInfo,
	currentPrice float64,
	pnlPct float64,
	state *BaselinePositionState,
	action string,
	data *market.Data,
	cfg *store.BaselineConfig,
) *decision.Decision {
	// 1. 强制止损（CRITICAL - 最高优先级）
	hardStopLossPct := cfg.RiskManagement.HardStopLossPct
	if hardStopLossPct <= 0 {
		hardStopLossPct = 3.0 // 默认 -3.0%
	}
	if currentPrice >= state.EntryPrice*(1+hardStopLossPct/100) {
		return &decision.Decision{
			Symbol:    pos.Symbol,
			Action:    action,
			Reasoning: "Baseline: Hard stop loss (CRITICAL)",
		}
	}

	// 2. 终极止盈：RSI < 30 或 StochRSI 金叉
	rsi7 := data.CurrentRSI7
	if e.config.Indicators.EnableRSI && rsi7 < 30 {
		return &decision.Decision{
			Symbol:    pos.Symbol,
			Action:    action,
			Reasoning: "Baseline: RSI oversold exit (<30)",
		}
	}

	k, d := e.getStochRSI(data)
	// StochRSI 出场：要求 K 值在超卖区（<=30）且金叉
	// 优化：提高触发门槛，减少频繁出场
	if e.config.Indicators.EnableStochRSI && k <= 30 && k > d {
		return &decision.Decision{
			Symbol:    pos.Symbol,
			Action:    action,
			Reasoning: "Baseline: StochRSI golden cross exit",
		}
	}

	// 3. 移动止盈
	if state.TrailingTP > 0 && currentPrice >= state.TrailingTP {
		return &decision.Decision{
			Symbol:    pos.Symbol,
			Action:    action,
			Reasoning: "Baseline: Trailing take profit triggered",
		}
	}

	// 4. 移动止损
	if pnlPct >= 3.0 && currentPrice >= state.TrailingStop {
		return &decision.Decision{
			Symbol:    pos.Symbol,
			Action:    action,
			Reasoning: "Baseline: Trailing stop loss triggered",
		}
	}

	return nil
}

// updatePositionState 更新持仓状态（峰值价格和移动止盈止损）
func (e *BaselineEngine) updatePositionState(pos decision.PositionInfo, currentPrice float64) {
	stateKey := pos.Symbol + "_" + pos.Side
	state, exists := e.positionStates[stateKey]
	if !exists {
		return
	}

	// 获取配置参数
	cfg := e.config.BaselineConfig
	if cfg == nil {
		return
	}

	pnlPct := pos.UnrealizedPnLPct

	// 获取配置参数（多头和空头共用）
	rm := cfg.RiskManagement
	tp1Pct := rm.TrailingTP1Pct
	tp1Lock := rm.TrailingTP1Lock
	tp2Pct := rm.TrailingTP2Pct
	tp2Lock := rm.TrailingTP2Lock
	tp3Pct := rm.TrailingTP3Pct
	tp3Lock := rm.TrailingTP3Lock
	sl1Pct := rm.TrailingSL1Pct
	sl1Lock := rm.TrailingSL1Lock
	sl2Pct := rm.TrailingSL2Pct
	sl2Lock := rm.TrailingSL2Lock

	// 设置默认值
	if tp1Pct <= 0 {
		tp1Pct = 2.0
	}
	if tp1Lock <= 0 {
		tp1Lock = 0.5
	}
	if tp2Pct <= 0 {
		tp2Pct = 4.0
	}
	if tp2Lock <= 0 {
		tp2Lock = 1.5
	}
	if tp3Pct <= 0 {
		tp3Pct = 8.0
	}
	if tp3Lock <= 0 {
		tp3Lock = 2.0
	}
	if sl1Pct <= 0 {
		sl1Pct = 3.0
	}
	if sl1Lock <= 0 {
		sl1Lock = 1.0
	}
	if sl2Pct <= 0 {
		sl2Pct = 5.0
	}
	if sl2Lock <= 0 {
		sl2Lock = 1.5
	}

	if pos.Side == "long" {
		// 更新峰值价格
		if currentPrice > state.PeakPrice {
			state.PeakPrice = currentPrice
		}

		// 多级移动止盈
		if pnlPct >= tp3Pct {
			state.TrailingTP = currentPrice * (1 - tp3Lock/100)
		} else if pnlPct >= tp2Pct {
			state.TrailingTP = currentPrice * (1 - tp2Lock/100)
		} else if pnlPct >= tp1Pct {
			state.TrailingTP = currentPrice * (1 - tp1Lock/100)
		}

		// 移动止损
		if pnlPct >= sl2Pct {
			newStop := currentPrice * (1 - sl2Lock/100)
			if newStop > state.TrailingStop {
				state.TrailingStop = newStop
			}
		} else if pnlPct >= sl1Pct {
			newStop := state.EntryPrice * (1 + sl1Lock/100)
			if newStop > state.TrailingStop {
				state.TrailingStop = newStop
			}
		}
	} else {
		// 空头：更新峰值价格（最低价）
		if currentPrice < state.PeakPrice || state.PeakPrice == state.EntryPrice {
			state.PeakPrice = currentPrice
		}

		// 多级移动止盈（使用配置参数，空头方向相反）
		if pnlPct >= tp3Pct {
			state.TrailingTP = currentPrice * (1 + tp3Lock/100)
		} else if pnlPct >= tp2Pct {
			state.TrailingTP = currentPrice * (1 + tp2Lock/100)
		} else if pnlPct >= tp1Pct {
			state.TrailingTP = currentPrice * (1 + tp1Lock/100)
		}

		// 移动止损（使用配置参数，空头方向相反）
		if pnlPct >= sl2Pct {
			newStop := currentPrice * (1 + sl2Lock/100)
			if newStop < state.TrailingStop {
				state.TrailingStop = newStop
			}
		} else if pnlPct >= sl1Pct {
			newStop := state.EntryPrice * (1 - sl1Lock/100)
			if newStop < state.TrailingStop {
				state.TrailingStop = newStop
			}
		}
	}
}

// 辅助方法

func (e *BaselineEngine) hasPosition(positions []decision.PositionInfo, symbol string) bool {
	for _, pos := range positions {
		if pos.Symbol == symbol {
			return true
		}
	}
	return false
}

// countSameDirectionPositions 统计同方向仓位数量
func (e *BaselineEngine) countSameDirectionPositions(side string) int {
	count := 0
	for _, state := range e.positionStates {
		if state.Side == side {
			count++
		}
	}
	return count
}

func (e *BaselineEngine) getATR(data *market.Data) float64 {
	if data.TimeframeData != nil {
		for _, tfData := range data.TimeframeData {
			if tfData.ATR14 > 0 {
				return tfData.ATR14
			}
		}
	}
	if data.IntradaySeries != nil && data.IntradaySeries.ATR14 > 0 {
		return data.IntradaySeries.ATR14
	}
	return 0
}

func (e *BaselineEngine) getStochRSI(data *market.Data) (k, d float64) {
	if data.TimeframeData != nil {
		for _, tfData := range data.TimeframeData {
			if len(tfData.StochRSI_K) > 0 && len(tfData.StochRSI_D) > 0 {
				k = tfData.StochRSI_K[len(tfData.StochRSI_K)-1]
				d = tfData.StochRSI_D[len(tfData.StochRSI_D)-1]
				return k, d
			}
		}
	}
	return 0, 0
}

// getVolumeRatio 获取当前成交量与平均成交量的比值
// 返回值 > 1 表示成交量高于平均，< 1 表示低于平均
// 返回 0 表示无法获取成交量数据
func (e *BaselineEngine) getVolumeRatio(data *market.Data) float64 {
	if data.TimeframeData == nil {
		return 0
	}

	// 遍历时间周期数据，优先使用较短周期的数据
	for _, tfData := range data.TimeframeData {
		if len(tfData.Klines) < 5 {
			continue
		}

		// 获取最近一根 K 线的成交量
		currentVolume := tfData.Klines[len(tfData.Klines)-1].Volume
		if currentVolume <= 0 {
			continue
		}

		// 计算前 N 根 K 线的平均成交量（排除最新一根）
		lookback := 20
		if len(tfData.Klines)-1 < lookback {
			lookback = len(tfData.Klines) - 1
		}
		if lookback < 3 {
			continue
		}

		var sumVolume float64
		for i := len(tfData.Klines) - 1 - lookback; i < len(tfData.Klines)-1; i++ {
			sumVolume += tfData.Klines[i].Volume
		}
		avgVolume := sumVolume / float64(lookback)

		if avgVolume > 0 {
			return currentVolume / avgVolume
		}
	}

	return 0
}

// generateScoredDecision 生成带评分的开仓决策
// 返回 nil 表示不满足开仓条件
func (e *BaselineEngine) generateScoredDecision(
	symbol string,
	data *market.Data,
	equity float64,
	available float64,
) *ScoredDecision {
	if data == nil {
		return nil
	}

	// 获取配置参数
	baselineCfg := e.config.BaselineConfig
	if baselineCfg == nil {
		return nil
	}

	indicators := e.config.Indicators
	price := data.CurrentPrice
	ema20 := data.CurrentEMA20

	// 计算信号强度和评分
	longSignals := 0
	shortSignals := 0
	longScore := 0.0
	shortScore := 0.0

	// MACD 不参与开仓方向判断，仅作为参考指标
	// （已移除 MACD 对 longSignals/shortSignals 的影响）

	// EMA 趋势信号及评分（降低权重：最高 20 分）
	if indicators.EnableEMA && ema20 > 0 {
		priceDiff := (price - ema20) / ema20 * 100
		if price > ema20 {
			longSignals++
			// 价格高于 EMA 越多，评分越高
			longScore += min(priceDiff*5, 20) // 最高 20 分
		} else {
			shortSignals++
			// 价格低于 EMA 越多，评分越高
			shortScore += min(-priceDiff*5, 20)
		}
	}

	// StochRSI 信号及评分（趋势跟随：最高 70 分，成为主导指标）
	// 使用配置参数
	stochOversold := baselineCfg.SignalThresholds.StochOversold
	stochOverbought := baselineCfg.SignalThresholds.StochOverbought
	if stochOversold <= 0 {
		stochOversold = 15 // 默认值（优化：从 20 收紧到 15）
	}
	if stochOverbought <= 0 {
		stochOverbought = 85 // 默认值（优化：从 80 收紧到 85）
	}

	k, d := e.getStochRSI(data)
	if indicators.EnableStochRSI && k > 0 && d > 0 {
		// 做多信号：金叉且脱离超卖区（趋势确认）
		if k > stochOversold && k > d && k < stochOverbought {
			longSignals++
			// K 值在中间区域（20-80）且金叉，评分越高
			// 位置评分：K值在40-60最佳，最高 35 分
			positionScore := 35.0
			if k < 40 {
				positionScore = (k - 20) / 20 * 35
			} else if k > 60 {
				positionScore = (80 - k) / 20 * 35
			}
			// 金叉强度评分：最高 35 分
			crossScore := min((k-d)/20*35, 35)
			longScore += positionScore + crossScore // 最高 70 分
		}
		// 做空信号：死叉且脱离超买区（趋势确认）
		if k < stochOverbought && k < d && k > stochOversold {
			shortSignals++
			// K 值在中间区域（20-80）且死叉，评分越高
			// 位置评分：K值在40-60最佳，最高 35 分
			positionScore := 35.0
			if k < 40 {
				positionScore = (k - 20) / 20 * 35
			} else if k > 60 {
				positionScore = (80 - k) / 20 * 35
			}
			// 死叉强度评分：最高 35 分
			crossScore := min((d-k)/20*35, 35)
			shortScore += positionScore + crossScore // 最高 70 分
		}
	}

	// 成交量确认：根据成交量比值调整评分
	// 成交量高于平均值时增加评分，低于平均值时降低评分
	if indicators.EnableVolume {
		volumeRatio := e.getVolumeRatio(data)
		if volumeRatio > 0 {
			// 成交量调整系数：0.7 ~ 1.3
			var volumeMultiplier float64
			if volumeRatio < 0.5 {
				// 成交量过低（<50% 平均值），大幅降低评分
				volumeMultiplier = 0.7
			} else if volumeRatio < 0.8 {
				// 成交量偏低（50%-80% 平均值），适度降低评分
				volumeMultiplier = 0.85
			} else if volumeRatio > 2.0 {
				// 成交量异常高（>200% 平均值），可能是异常波动，不加分
				volumeMultiplier = 1.0
			} else if volumeRatio > 1.5 {
				// 成交量较高（150%-200% 平均值），增加评分
				volumeMultiplier = 1.2
			} else if volumeRatio > 1.2 {
				// 成交量略高（120%-150% 平均值），略微增加评分
				volumeMultiplier = 1.1
			} else {
				// 正常成交量（80%-120% 平均值）
				volumeMultiplier = 1.0
			}
			longScore *= volumeMultiplier
			shortScore *= volumeMultiplier
		}
	}

	// 检查是否满足最小信号数要求
	minSignals := baselineCfg.SignalThresholds.MinSignalCount
	if minSignals <= 0 {
		minSignals = 3 // 默认值（优化：从 2 提高到 3）
	}

	// 计算仓位参数
	leverage := baselineCfg.RiskManagement.Leverage
	if leverage <= 0 {
		leverage = 5
	}
	maxPos := e.config.RiskControl.MaxPositions
	if maxPos <= 0 {
		maxPos = 3
	}
	positionValue := (available / float64(maxPos)) * float64(leverage)
	if positionValue < 50 {
		return nil
	}

	hardStopLossPct := baselineCfg.RiskManagement.HardStopLossPct
	if hardStopLossPct <= 0 {
		hardStopLossPct = 3.0 // 默认 -3.0%
	}

	// 获取同方向最大仓位数限制
	maxSameDir := baselineCfg.RiskManagement.MaxSameDirectionPositions
	if maxSameDir <= 0 {
		maxSameDir = 2 // 默认最多 2 个同方向仓位
	}

	// 生成做多决策
	if longSignals >= minSignals && longScore > 0 {
		// 检查同方向仓位数量限制
		if e.countSameDirectionPositions("long") >= maxSameDir {
			return nil
		}

		// 检查是否已有相同币种的持仓状态
		stateKey := symbol + "_long"
		if _, exists := e.positionStates[stateKey]; exists {
			return nil
		}

		stopLossPrice := price * (1 - hardStopLossPct/100)
		e.positionStates[stateKey] = &BaselinePositionState{
			Symbol:        symbol,
			Side:          "long",
			EntryPrice:    price,
			PeakPrice:     price,
			TrailingStop:  stopLossPrice,
			TrailingTP:    0,
			HardStopPrice: stopLossPrice, // 挂单止损价
		}

		return &ScoredDecision{
			Decision: decision.Decision{
				Symbol:          symbol,
				Action:          "open_long",
				Leverage:        leverage,
				PositionSizeUSD: positionValue,
				StopLoss:        stopLossPrice,
				TakeProfit:      0,
				Confidence:      75,
				Reasoning:       "Baseline: Multiple long signals",
			},
			Score: longScore,
		}
	}

	// 生成做空决策
	if shortSignals >= minSignals && shortScore > 0 {
		// 检查同方向仓位数量限制
		if e.countSameDirectionPositions("short") >= maxSameDir {
			return nil
		}

		// 检查是否已有相同币种的持仓状态
		stateKey := symbol + "_short"
		if _, exists := e.positionStates[stateKey]; exists {
			return nil
		}

		stopLossPrice := price * (1 + hardStopLossPct/100)
		e.positionStates[stateKey] = &BaselinePositionState{
			Symbol:        symbol,
			Side:          "short",
			EntryPrice:    price,
			PeakPrice:     price,
			TrailingStop:  stopLossPrice,
			TrailingTP:    0,
			HardStopPrice: stopLossPrice, // 挂单止损价
		}

		return &ScoredDecision{
			Decision: decision.Decision{
				Symbol:          symbol,
				Action:          "open_short",
				Leverage:        leverage,
				PositionSizeUSD: positionValue,
				StopLoss:        stopLossPrice,
				TakeProfit:      0,
				Confidence:      75,
				Reasoning:       "Baseline: Multiple short signals",
			},
			Score: shortScore,
		}
	}

	return nil
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// selectBestDecisions 根据评分筛选最优的开仓决策
// currentPositions: 当前持仓数量
// 返回: 筛选后的决策列表（不超过 max_positions 限制）
func (e *BaselineEngine) selectBestDecisions(
	candidates []ScoredDecision,
	currentPositions int,
) []decision.Decision {
	if len(candidates) == 0 {
		return []decision.Decision{}
	}

	// 计算可开仓数量
	maxPositions := e.config.RiskControl.MaxPositions
	if maxPositions <= 0 {
		maxPositions = 3
	}
	availableSlots := maxPositions - currentPositions
	if availableSlots <= 0 {
		return []decision.Decision{}
	}

	// 按评分从高到低排序
	sortedCandidates := make([]ScoredDecision, len(candidates))
	copy(sortedCandidates, candidates)

	// 简单的冒泡排序（因为候选数量通常不多）
	for i := 0; i < len(sortedCandidates)-1; i++ {
		for j := 0; j < len(sortedCandidates)-i-1; j++ {
			if sortedCandidates[j].Score < sortedCandidates[j+1].Score {
				sortedCandidates[j], sortedCandidates[j+1] = sortedCandidates[j+1], sortedCandidates[j]
			}
		}
	}

	// 选择评分最高的前 N 个决策
	selectedCount := availableSlots
	if len(sortedCandidates) < selectedCount {
		selectedCount = len(sortedCandidates)
	}

	result := make([]decision.Decision, selectedCount)
	for i := 0; i < selectedCount; i++ {
		result[i] = sortedCandidates[i].Decision
	}

	return result
}

// CheckPendingStopLoss 检查挂单止损是否触发（基于OHLC数据）
// 返回需要止损平仓的决策列表
func (e *BaselineEngine) CheckPendingStopLoss(
	marketData map[string]*market.Data,
	positions []decision.PositionInfo,
) []decision.Decision {
	stopDecisions := make([]decision.Decision, 0)

	for _, pos := range positions {
		stateKey := pos.Symbol + "_" + pos.Side
		state, exists := e.positionStates[stateKey]
		if !exists || state.HardStopPrice <= 0 {
			logger.Debugf("[Baseline] %s: no state or HardStopPrice=0", stateKey)
			continue
		}

		data, ok := marketData[pos.Symbol]
		if !ok || data == nil {
			continue
		}

		// 获取当前 bar 的 OHLC 数据
		barLow, barHigh := e.getCurrentBarOHLC(data)
		if barLow <= 0 || barHigh <= 0 {
			logger.Debugf("[Baseline] %s: OHLC not available (low=%.2f, high=%.2f)", pos.Symbol, barLow, barHigh)
			continue
		}

		logger.Debugf("[Baseline] %s %s: entry=%.4f, stopPrice=%.4f, barHigh=%.4f, barLow=%.4f",
			pos.Symbol, pos.Side, state.EntryPrice, state.HardStopPrice, barHigh, barLow)

		triggered := false
		action := ""

		if pos.Side == "long" {
			// 多头：检查 bar 最低价是否触及止损价
			if barLow <= state.HardStopPrice {
				triggered = true
				action = "close_long"
			}
		} else {
			// 空头：检查 bar 最高价是否触及止损价
			if barHigh >= state.HardStopPrice {
				triggered = true
				action = "close_short"
			}
		}

		if triggered {
			stopDecisions = append(stopDecisions, decision.Decision{
				Symbol:    pos.Symbol,
				Action:    action,
				Reasoning: "Baseline: Pending stop loss triggered (OHLC)",
			})
			// 清除持仓状态
			delete(e.positionStates, stateKey)
		}
	}

	return stopDecisions
}

// getCurrentBarOHLC 获取当前 bar 的最低价和最高价
func (e *BaselineEngine) getCurrentBarOHLC(data *market.Data) (low, high float64) {
	// 直接使用 market.Data 中的 OHLC 数据
	// 这些数据来自 BuildDataFromKlines，代表当前 bar 的真实 OHLC
	if data.Low > 0 && data.High > 0 {
		return data.Low, data.High
	}

	// 如果主数据没有，尝试从 TimeframeData 获取
	if data.TimeframeData != nil {
		for _, tfData := range data.TimeframeData {
			if len(tfData.Klines) == 0 {
				continue
			}
			// 获取最新一根 K 线
			lastBar := tfData.Klines[len(tfData.Klines)-1]
			if lastBar.Low > 0 && lastBar.High > 0 {
				return lastBar.Low, lastBar.High
			}
		}
	}

	return 0, 0
}
