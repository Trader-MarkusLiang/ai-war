import { useState, useEffect } from 'react'
import { Activity, TrendingUp, Shield, AlertCircle, PlayCircle } from 'lucide-react'
import type { BaselineConfig, StrategyConfig } from '../../types'

interface BaselineConfigEditorProps {
  config: BaselineConfig | null
  strategyConfig: StrategyConfig
  onChange: (config: BaselineConfig) => void
  language: 'zh' | 'en'
}

const DEFAULT_BASELINE_CONFIG: BaselineConfig = {
  rsi_period: 14,
  macd_fast: 12,
  macd_slow: 26,
  macd_signal: 9,
  ema_period: 20,
  stoch_rsi_period: 14,
  atr_period: 14,
  signal_thresholds: {
    rsi_oversold: 30,
    rsi_overbought: 70,
    stoch_oversold: 20,
    stoch_overbought: 80,
    min_signal_count: 2,
  },
  risk_management: {
    equity_multiplier: 5.0,
    leverage: 5,
    hard_stop_loss_pct: 2.0,
    trailing_tp1_pct: 2.0,
    trailing_tp1_lock: 0.5,
    trailing_tp2_pct: 4.0,
    trailing_tp2_lock: 1.0,
    trailing_tp3_pct: 6.0,
    trailing_tp3_lock: 1.5,
    trailing_sl1_pct: 3.0,
    trailing_sl1_lock: 1.0,
    trailing_sl2_pct: 5.0,
    trailing_sl2_lock: 1.5,
  },
}

export function BaselineConfigEditor({
  config,
  strategyConfig,
  onChange,
  language,
}: BaselineConfigEditorProps) {
  const [localConfig, setLocalConfig] = useState<BaselineConfig>(
    config || DEFAULT_BASELINE_CONFIG
  )

  useEffect(() => {
    if (config) {
      setLocalConfig(config)
    }
  }, [config])

  const aiIndicators = strategyConfig.indicators
  const aiRiskControl = strategyConfig.risk_control

  const handleChange = (updates: Partial<BaselineConfig>) => {
    const newConfig = { ...localConfig, ...updates }
    setLocalConfig(newConfig)
    onChange(newConfig)
  }

  const handleThresholdChange = (key: keyof BaselineConfig['signal_thresholds'], value: number) => {
    const newConfig = {
      ...localConfig,
      signal_thresholds: {
        ...localConfig.signal_thresholds,
        [key]: value,
      },
    }
    setLocalConfig(newConfig)
    onChange(newConfig)
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2 pb-4 border-b" style={{ borderColor: '#2B3139' }}>
        <Activity className="w-5 h-5" style={{ color: '#FF6B35' }} />
        <h3 className="text-lg font-semibold" style={{ color: '#EAECEF' }}>
          {language === 'zh' ? 'Baseline 策略配置' : 'Baseline Strategy Configuration'}
        </h3>
      </div>

      <div className="p-3 rounded-lg" style={{ background: 'rgba(255, 107, 53, 0.1)', border: '1px solid rgba(255, 107, 53, 0.3)' }}>
        <p className="text-xs" style={{ color: '#848E9C' }}>
          {language === 'zh'
            ? '配置传统技术指标策略参数，用于与 AI 策略进行对比。Baseline 策略使用 RSI、MACD、EMA、StochRSI 等指标的信号共振来做出交易决策。'
            : 'Configure traditional technical indicator strategy parameters for comparison with AI strategy. Baseline uses signal confluence from RSI, MACD, EMA, StochRSI indicators.'}
        </p>
      </div>
      {/* Baseline Trading Rules Explanation */}
      <div className="p-4 rounded-lg space-y-3" style={{ background: 'rgba(16, 185, 129, 0.1)', border: '1px solid rgba(16, 185, 129, 0.3)' }}>
        <div className="flex items-center gap-2">
          <PlayCircle className="w-5 h-5" style={{ color: '#10B981' }} />
          <h4 className="font-semibold" style={{ color: '#EAECEF' }}>
            {language === 'zh' ? 'Baseline 交易规则说明' : 'Baseline Trading Rules'}
          </h4>
        </div>

        <div className="space-y-2 text-sm" style={{ color: '#848E9C' }}>
          <div>
            <span className="font-medium" style={{ color: '#10B981' }}>
              {language === 'zh' ? '入场规则：' : 'Entry Rules: '}
            </span>
            {language === 'zh'
              ? '需要至少 2 个指标信号共振才开仓。做多信号：RSI<30(超卖) + MACD>0 + 价格>EMA20 + StochRSI金叉。做空信号：RSI>70(超买) + MACD<0 + 价格<EMA20 + StochRSI死叉。'
              : 'Requires ≥2 indicator signals confluence. Long: RSI<30(oversold) + MACD>0 + Price>EMA20 + StochRSI golden cross. Short: RSI>70(overbought) + MACD<0 + Price<EMA20 + StochRSI death cross.'}
          </div>

          <div>
            <span className="font-medium" style={{ color: '#10B981' }}>
              {language === 'zh' ? '出场规则：' : 'Exit Rules: '}
            </span>
            {language === 'zh'
              ? '硬止损 -2% (最高优先级)，多级移动止盈 (2%/4%/6% 阶梯)，移动止损 (3% 后锁定 1% 利润)，或 RSI/StochRSI 反转信号。'
              : 'Hard stop -2% (highest priority), multi-level trailing TP (2%/4%/6% tiers), trailing SL (lock 1% profit after 3% gain), or RSI/StochRSI reversal signals.'}
          </div>

          <div>
            <span className="font-medium" style={{ color: '#10B981' }}>
              {language === 'zh' ? '仓位管理：' : 'Position Management: '}
            </span>
            {language === 'zh'
              ? '动态计算仓位 = (可用资金 / 最大持仓数) × 杠杆，统一 5x 杠杆，总仓位限制为本金 90%，最大持仓数与 AI 策略共享。'
              : 'Dynamic position sizing = (available / max positions) × leverage, unified 5x leverage, total position limit 90% of principal, max positions shared with AI strategy.'}
          </div>
        </div>

        <div className="p-2 rounded" style={{ background: 'rgba(16, 185, 129, 0.1)', border: '1px solid rgba(16, 185, 129, 0.3)' }}>
          <p className="text-xs" style={{ color: '#10B981' }}>
            {language === 'zh'
              ? '✅ 已对齐 AI 策略：-2% 硬止损、多级移动止盈止损、动态仓位管理、统一 5x 杠杆。'
              : '✅ Aligned with AI strategy: -2% hard stop, multi-level trailing TP/SL, dynamic position sizing, unified 5x leverage.'}
          </p>
        </div>
      </div>

      {/* Decision Preview Section */}
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          <AlertCircle className="w-5 h-5" style={{ color: '#3B82F6' }} />
          <h4 className="font-semibold" style={{ color: '#EAECEF' }}>
            {language === 'zh' ? '决策预览' : 'Decision Preview'}
          </h4>
        </div>

        {/* Example 1: Long Entry */}
        <div className="p-4 rounded-lg space-y-2" style={{ background: 'rgba(59, 130, 246, 0.1)', border: '1px solid rgba(59, 130, 246, 0.3)' }}>
          <div className="flex items-center justify-between">
            <span className="font-medium" style={{ color: '#3B82F6' }}>
              {language === 'zh' ? '示例 1: 做多入场' : 'Example 1: Long Entry'}
            </span>
            <span className="text-xs px-2 py-1 rounded" style={{ background: 'rgba(16, 185, 129, 0.2)', color: '#10B981' }}>
              {language === 'zh' ? '开多' : 'OPEN LONG'}
            </span>
          </div>
          <div className="text-sm space-y-1" style={{ color: '#848E9C' }}>
            <div><span style={{ color: '#EAECEF' }}>{language === 'zh' ? '交易对：' : 'Symbol: '}</span>BTCUSDT @ $42,500</div>
            <div><span style={{ color: '#EAECEF' }}>{language === 'zh' ? '市场状况：' : 'Market: '}</span>
              {language === 'zh' ? 'RSI=28(超卖), MACD=+15(金叉), 价格>EMA20, StochRSI金叉' : 'RSI=28(oversold), MACD=+15(bullish), Price>EMA20, StochRSI golden cross'}
            </div>
            <div><span style={{ color: '#EAECEF' }}>{language === 'zh' ? '信号分析：' : 'Signals: '}</span>
              {language === 'zh' ? '4个做多信号共振 ✓' : '4 long signals confluence ✓'}
            </div>
            <div><span style={{ color: '#EAECEF' }}>{language === 'zh' ? '执行：' : 'Execution: '}</span>
              {language === 'zh' ? '开多 5x 杠杆，仓位 $1,667 (可用资金1000÷3×5)，硬止损 $41,650(-2%)，移动止盈启动' : 'Open long 5x leverage, position $1,667 (available 1000÷3×5), hard SL $41,650(-2%), trailing TP activated'}
            </div>
          </div>
        </div>

        {/* Example 2: Short Entry */}
        <div className="p-4 rounded-lg space-y-2" style={{ background: 'rgba(239, 68, 68, 0.1)', border: '1px solid rgba(239, 68, 68, 0.3)' }}>
          <div className="flex items-center justify-between">
            <span className="font-medium" style={{ color: '#EF4444' }}>
              {language === 'zh' ? '示例 2: 做空入场' : 'Example 2: Short Entry'}
            </span>
            <span className="text-xs px-2 py-1 rounded" style={{ background: 'rgba(239, 68, 68, 0.2)', color: '#EF4444' }}>
              {language === 'zh' ? '开空' : 'OPEN SHORT'}
            </span>
          </div>
          <div className="text-sm space-y-1" style={{ color: '#848E9C' }}>
            <div><span style={{ color: '#EAECEF' }}>{language === 'zh' ? '交易对：' : 'Symbol: '}</span>ETHUSDT @ $2,250</div>
            <div><span style={{ color: '#EAECEF' }}>{language === 'zh' ? '市场状况：' : 'Market: '}</span>
              {language === 'zh' ? 'RSI=75(超买), MACD=-8(死叉), 价格<EMA20, StochRSI死叉' : 'RSI=75(overbought), MACD=-8(bearish), Price<EMA20, StochRSI death cross'}
            </div>
            <div><span style={{ color: '#EAECEF' }}>{language === 'zh' ? '信号分析：' : 'Signals: '}</span>
              {language === 'zh' ? '4个做空信号共振 ✓' : '4 short signals confluence ✓'}
            </div>
            <div><span style={{ color: '#EAECEF' }}>{language === 'zh' ? '执行：' : 'Execution: '}</span>
              {language === 'zh' ? '开空 5x 杠杆，仓位 $1,111 (可用资金667÷3×5)，硬止损 $2,295(+2%)，移动止盈启动' : 'Open short 5x leverage, position $1,111 (available 667÷3×5), hard SL $2,295(+2%), trailing TP activated'}
            </div>
          </div>
        </div>

        {/* Example 3: Exit Decision */}
        <div className="p-4 rounded-lg space-y-2" style={{ background: 'rgba(245, 158, 11, 0.1)', border: '1px solid rgba(245, 158, 11, 0.3)' }}>
          <div className="flex items-center justify-between">
            <span className="font-medium" style={{ color: '#F59E0B' }}>
              {language === 'zh' ? '示例 3: 出场决策' : 'Example 3: Exit Decision'}
            </span>
            <span className="text-xs px-2 py-1 rounded" style={{ background: 'rgba(245, 158, 11, 0.2)', color: '#F59E0B' }}>
              {language === 'zh' ? '平仓' : 'CLOSE'}
            </span>
          </div>
          <div className="text-sm space-y-1" style={{ color: '#848E9C' }}>
            <div><span style={{ color: '#EAECEF' }}>{language === 'zh' ? '持仓：' : 'Position: '}</span>
              {language === 'zh' ? 'BTCUSDT 多头，入场 $42,500，当前 $43,800' : 'BTCUSDT long, entry $42,500, current $43,800'}
            </div>
            <div><span style={{ color: '#EAECEF' }}>{language === 'zh' ? '盈亏：' : 'PnL: '}</span>
              <span style={{ color: '#10B981' }}>+3.06%</span>
            </div>
            <div><span style={{ color: '#EAECEF' }}>{language === 'zh' ? '市场状况：' : 'Market: '}</span>
              {language === 'zh' ? 'RSI=72(超买区域)' : 'RSI=72(overbought zone)'}
            </div>
            <div><span style={{ color: '#EAECEF' }}>{language === 'zh' ? '决策：' : 'Decision: '}</span>
              {language === 'zh' ? '平多仓 - 移动止损触发 (锁定 1% 利润)' : 'Close long - Trailing SL triggered (1% profit locked)'}
            </div>
          </div>
        </div>
      </div>

      {/* AI Strategy Parameters Comparison */}
      <div className="p-4 rounded-lg space-y-3" style={{ background: 'rgba(59, 130, 246, 0.1)', border: '1px solid rgba(59, 130, 246, 0.3)' }}>
        <div className="flex items-center gap-2">
          <TrendingUp className="w-5 h-5" style={{ color: '#3B82F6' }} />
          <h4 className="font-semibold" style={{ color: '#EAECEF' }}>
            {language === 'zh' ? 'AI 策略参数（参考）' : 'AI Strategy Parameters (Reference)'}
          </h4>
        </div>

        <div className="space-y-3 text-sm">
          {/* Timeframe Configuration */}
          <div className="p-3 rounded" style={{ background: 'rgba(59, 130, 246, 0.1)', border: '1px solid rgba(59, 130, 246, 0.2)' }}>
            <div className="text-xs mb-2 font-medium" style={{ color: '#3B82F6' }}>
              {language === 'zh' ? '📊 时间周期配置' : '📊 Timeframe Configuration'}
            </div>
            <div className="space-y-1" style={{ color: '#EAECEF' }}>
              <div>
                <span style={{ color: '#848E9C' }}>{language === 'zh' ? '主周期：' : 'Primary: '}</span>
                {aiIndicators.klines?.primary_timeframe || '4h'}
                {aiIndicators.klines?.enable_multi_timeframe && aiIndicators.klines?.longer_timeframe && (
                  <span style={{ color: '#848E9C' }}> + {aiIndicators.klines.longer_timeframe}</span>
                )}
              </div>
              <div>
                <span style={{ color: '#848E9C' }}>{language === 'zh' ? '主周期K线数：' : 'Primary K-lines: '}</span>
                {aiIndicators.klines?.primary_count || 35} {language === 'zh' ? '根' : 'bars'}
              </div>
              {aiIndicators.klines?.enable_multi_timeframe && aiIndicators.klines?.longer_timeframe && (
                <div>
                  <span style={{ color: '#848E9C' }}>{language === 'zh' ? '长周期K线数：' : 'Longer K-lines: '}</span>
                  {aiIndicators.klines?.longer_count || 100} {language === 'zh' ? '根' : 'bars'}
                </div>
              )}
            </div>
          </div>

          {/* Indicator Configuration */}
          <div className="grid grid-cols-2 gap-3">
            <div className="p-3 rounded" style={{ background: 'rgba(59, 130, 246, 0.1)', border: '1px solid rgba(59, 130, 246, 0.2)' }}>
              <div className="text-xs mb-2 font-medium" style={{ color: '#3B82F6' }}>
                {language === 'zh' ? '指标配置' : 'Indicators'}
              </div>
              <div className="space-y-1" style={{ color: '#EAECEF', fontSize: '0.8rem' }}>
                <div>RSI: {aiIndicators.enable_rsi ? '✓' : '✗'} {aiIndicators.rsi_periods && aiIndicators.rsi_periods.length > 0 ? `(${aiIndicators.rsi_periods.join(', ')})` : ''}</div>
                <div>MACD: {aiIndicators.enable_macd ? '✓' : '✗'} {aiIndicators.macd_fast_period ? `(${aiIndicators.macd_fast_period},${aiIndicators.macd_slow_period},${aiIndicators.macd_signal_period})` : ''}</div>
                <div>EMA: {aiIndicators.enable_ema ? '✓' : '✗'} {aiIndicators.ema_periods && aiIndicators.ema_periods.length > 0 ? `(${aiIndicators.ema_periods.join(', ')})` : ''}</div>
                <div>StochRSI: {aiIndicators.enable_stoch_rsi ? '✓' : '✗'} {aiIndicators.stoch_rsi_length_rsi ? `(${aiIndicators.stoch_rsi_length_rsi})` : ''}</div>
              </div>
            </div>

            <div className="p-3 rounded" style={{ background: 'rgba(59, 130, 246, 0.1)', border: '1px solid rgba(59, 130, 246, 0.2)' }}>
              <div className="text-xs mb-2 font-medium" style={{ color: '#3B82F6' }}>
                {language === 'zh' ? '风险控制' : 'Risk Control'}
              </div>
              <div className="space-y-1" style={{ color: '#EAECEF', fontSize: '0.8rem' }}>
                <div>{language === 'zh' ? 'BTC/ETH 杠杆' : 'BTC/ETH Leverage'}: {aiRiskControl.btc_eth_max_leverage || 5}x</div>
                <div>{language === 'zh' ? '山寨币杠杆' : 'Altcoin Leverage'}: {aiRiskControl.altcoin_max_leverage || 5}x</div>
                <div>{language === 'zh' ? '最大持仓数' : 'Max Positions'}: {aiRiskControl.max_positions || 3}</div>
                <div>{language === 'zh' ? '总仓位限制' : 'Total Position Limit'}: {aiRiskControl.max_total_position_pct || 90}%</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Indicator Parameters */}
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          <Activity className="w-5 h-5" style={{ color: '#FF6B35' }} />
          <h4 className="font-semibold" style={{ color: '#EAECEF' }}>
            {language === 'zh' ? '指标参数配置' : 'Indicator Parameters'}
          </h4>
        </div>

        <div className="grid grid-cols-2 gap-4">
          {/* RSI Period */}
          <div>
            <label className="block text-sm mb-2" style={{ color: '#848E9C' }}>
              {language === 'zh' ? 'RSI 周期' : 'RSI Period'}
            </label>
            <input
              type="number"
              value={localConfig.rsi_period}
              onChange={(e) => handleChange({ rsi_period: Number(e.target.value) })}
              className="w-full px-3 py-2 rounded"
              style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
              min={7}
              max={50}
            />
          </div>

          {/* MACD Fast */}
          <div>
            <label className="block text-sm mb-2" style={{ color: '#848E9C' }}>
              {language === 'zh' ? 'MACD 快线' : 'MACD Fast'}
            </label>
            <input
              type="number"
              value={localConfig.macd_fast}
              onChange={(e) => handleChange({ macd_fast: Number(e.target.value) })}
              className="w-full px-3 py-2 rounded"
              style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
              min={5}
              max={20}
            />
          </div>

          {/* MACD Slow */}
          <div>
            <label className="block text-sm mb-2" style={{ color: '#848E9C' }}>
              {language === 'zh' ? 'MACD 慢线' : 'MACD Slow'}
            </label>
            <input
              type="number"
              value={localConfig.macd_slow}
              onChange={(e) => handleChange({ macd_slow: Number(e.target.value) })}
              className="w-full px-3 py-2 rounded"
              style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
              min={20}
              max={40}
            />
          </div>

          {/* MACD Signal */}
          <div>
            <label className="block text-sm mb-2" style={{ color: '#848E9C' }}>
              {language === 'zh' ? 'MACD 信号线' : 'MACD Signal'}
            </label>
            <input
              type="number"
              value={localConfig.macd_signal}
              onChange={(e) => handleChange({ macd_signal: Number(e.target.value) })}
              className="w-full px-3 py-2 rounded"
              style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
              min={5}
              max={15}
            />
          </div>

          {/* EMA Period */}
          <div>
            <label className="block text-sm mb-2" style={{ color: '#848E9C' }}>
              {language === 'zh' ? 'EMA 周期' : 'EMA Period'}
            </label>
            <input
              type="number"
              value={localConfig.ema_period}
              onChange={(e) => handleChange({ ema_period: Number(e.target.value) })}
              className="w-full px-3 py-2 rounded"
              style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
              min={10}
              max={50}
            />
          </div>

          {/* StochRSI Period */}
          <div>
            <label className="block text-sm mb-2" style={{ color: '#848E9C' }}>
              {language === 'zh' ? 'StochRSI 周期' : 'StochRSI Period'}
            </label>
            <input
              type="number"
              value={localConfig.stoch_rsi_period}
              onChange={(e) => handleChange({ stoch_rsi_period: Number(e.target.value) })}
              className="w-full px-3 py-2 rounded"
              style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
              min={7}
              max={30}
            />
          </div>

          {/* ATR Period */}
          <div>
            <label className="block text-sm mb-2" style={{ color: '#848E9C' }}>
              {language === 'zh' ? 'ATR 周期' : 'ATR Period'}
            </label>
            <input
              type="number"
              value={localConfig.atr_period}
              onChange={(e) => handleChange({ atr_period: Number(e.target.value) })}
              className="w-full px-3 py-2 rounded"
              style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
              min={7}
              max={30}
            />
          </div>
        </div>
      </div>

      {/* Signal Thresholds */}
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          <TrendingUp className="w-5 h-5" style={{ color: '#FF6B35' }} />
          <h4 className="font-semibold" style={{ color: '#EAECEF' }}>
            {language === 'zh' ? '信号阈值配置' : 'Signal Thresholds'}
          </h4>
        </div>

        <div className="grid grid-cols-2 gap-4">
          {/* RSI Oversold */}
          <div>
            <label className="block text-sm mb-2" style={{ color: '#848E9C' }}>
              {language === 'zh' ? 'RSI 超卖阈值' : 'RSI Oversold'}
            </label>
            <input
              type="number"
              value={localConfig.signal_thresholds.rsi_oversold}
              onChange={(e) => handleThresholdChange('rsi_oversold', Number(e.target.value))}
              className="w-full px-3 py-2 rounded"
              style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
              min={20}
              max={40}
            />
          </div>

          {/* RSI Overbought */}
          <div>
            <label className="block text-sm mb-2" style={{ color: '#848E9C' }}>
              {language === 'zh' ? 'RSI 超买阈值' : 'RSI Overbought'}
            </label>
            <input
              type="number"
              value={localConfig.signal_thresholds.rsi_overbought}
              onChange={(e) => handleThresholdChange('rsi_overbought', Number(e.target.value))}
              className="w-full px-3 py-2 rounded"
              style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
              min={60}
              max={80}
            />
          </div>

          {/* Stoch Oversold */}
          <div>
            <label className="block text-sm mb-2" style={{ color: '#848E9C' }}>
              {language === 'zh' ? 'StochRSI 超卖阈值' : 'StochRSI Oversold'}
            </label>
            <input
              type="number"
              value={localConfig.signal_thresholds.stoch_oversold}
              onChange={(e) => handleThresholdChange('stoch_oversold', Number(e.target.value))}
              className="w-full px-3 py-2 rounded"
              style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
              min={10}
              max={30}
            />
          </div>

          {/* Stoch Overbought */}
          <div>
            <label className="block text-sm mb-2" style={{ color: '#848E9C' }}>
              {language === 'zh' ? 'StochRSI 超买阈值' : 'StochRSI Overbought'}
            </label>
            <input
              type="number"
              value={localConfig.signal_thresholds.stoch_overbought}
              onChange={(e) => handleThresholdChange('stoch_overbought', Number(e.target.value))}
              className="w-full px-3 py-2 rounded"
              style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
              min={70}
              max={90}
            />
          </div>

          {/* Min Signal Count */}
          <div>
            <label className="block text-sm mb-2" style={{ color: '#848E9C' }}>
              {language === 'zh' ? '最小信号共振数' : 'Min Signal Count'}
            </label>
            <input
              type="number"
              value={localConfig.signal_thresholds.min_signal_count}
              onChange={(e) => handleThresholdChange('min_signal_count', Number(e.target.value))}
              className="w-full px-3 py-2 rounded"
              style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
              min={1}
              max={4}
            />
          </div>
        </div>
      </div>

      {/* Risk Management - Read Only Display */}
      <div className="space-y-4">
        <div className="flex items-center gap-2">
          <Shield className="w-5 h-5" style={{ color: '#FF6B35' }} />
          <h4 className="font-semibold" style={{ color: '#EAECEF' }}>
            {language === 'zh' ? '风险管理参数 (当前硬编码)' : 'Risk Management (Currently Hardcoded)'}
          </h4>
        </div>

        <div className="p-3 rounded-lg" style={{ background: 'rgba(255, 107, 53, 0.1)', border: '1px solid rgba(255, 107, 53, 0.3)' }}>
          <p className="text-xs mb-3" style={{ color: '#848E9C' }}>
            {language === 'zh'
              ? '以下参数当前为硬编码值，用于确保 Baseline 策略与 AI 策略的一致性。这些参数已在代码中固定，暂不支持修改。'
              : 'The following parameters are currently hardcoded to ensure consistency between Baseline and AI strategies. These values are fixed in code and not editable.'}
          </p>
        </div>

        {/* Position Sizing Section */}
        <div className="space-y-3">
          <div className="text-sm font-medium" style={{ color: '#10B981' }}>
            {language === 'zh' ? '📊 仓位管理' : '📊 Position Sizing'}
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div className="p-3 rounded" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
              <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '仓位计算公式' : 'Position Formula'}
              </div>
              <div className="text-sm font-semibold" style={{ color: '#EAECEF' }}>
                (可用/最大持仓数)×杠杆
              </div>
              <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '动态调整,确保资金充足' : 'Dynamic sizing, ensures sufficient funds'}
              </div>
            </div>

            <div className="p-3 rounded" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
              <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '杠杆倍数' : 'Leverage'}
              </div>
              <div className="text-lg font-semibold" style={{ color: '#EAECEF' }}>
                {localConfig.risk_management.leverage}x
              </div>
              <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '统一 5x 杠杆' : 'Unified 5x leverage'}
              </div>
            </div>

            <div className="p-3 rounded" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
              <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '总仓位限制' : 'Total Position Limit'}
              </div>
              <div className="text-lg font-semibold" style={{ color: '#EAECEF' }}>
                90%
              </div>
              <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '保留 10% 保证金缓冲' : 'Keep 10% margin buffer'}
              </div>
            </div>
          </div>
        </div>

        {/* Hard Stop Loss Section */}
        <div className="space-y-3">
          <div className="text-sm font-medium" style={{ color: '#EF4444' }}>
            {language === 'zh' ? '🛑 硬止损 (最高优先级)' : '🛑 Hard Stop Loss (Highest Priority)'}
          </div>
          <div className="p-3 rounded" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
            <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
              {language === 'zh' ? '硬止损百分比' : 'Hard Stop Loss %'}
            </div>
            <div className="text-lg font-semibold" style={{ color: '#EF4444' }}>
              -{localConfig.risk_management.hard_stop_loss_pct}%
            </div>
            <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
              {language === 'zh' ? '触发后立即平仓' : 'Immediate close on trigger'}
            </div>
          </div>
        </div>

        {/* Trailing Take Profit Section */}
        <div className="space-y-3">
          <div className="text-sm font-medium" style={{ color: '#10B981' }}>
            {language === 'zh' ? '📈 多级移动止盈' : '📈 Multi-level Trailing Take Profit'}
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div className="p-3 rounded" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
              <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '第 1 档' : 'Tier 1'}
              </div>
              <div className="text-sm font-semibold" style={{ color: '#10B981' }}>
                {localConfig.risk_management.trailing_tp1_pct}% → {localConfig.risk_management.trailing_tp1_lock}%
              </div>
              <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '盈利 2% 锁定 0.5%' : 'Profit 2% locks 0.5%'}
              </div>
            </div>

            <div className="p-3 rounded" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
              <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '第 2 档' : 'Tier 2'}
              </div>
              <div className="text-sm font-semibold" style={{ color: '#10B981' }}>
                {localConfig.risk_management.trailing_tp2_pct}% → {localConfig.risk_management.trailing_tp2_lock}%
              </div>
              <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '盈利 4% 锁定 1%' : 'Profit 4% locks 1%'}
              </div>
            </div>

            <div className="p-3 rounded" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
              <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '第 3 档' : 'Tier 3'}
              </div>
              <div className="text-sm font-semibold" style={{ color: '#10B981' }}>
                {localConfig.risk_management.trailing_tp3_pct}% → {localConfig.risk_management.trailing_tp3_lock}%
              </div>
              <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '盈利 6% 锁定 1.5%' : 'Profit 6% locks 1.5%'}
              </div>
            </div>
          </div>
        </div>

        {/* Trailing Stop Loss Section */}
        <div className="space-y-3">
          <div className="text-sm font-medium" style={{ color: '#F59E0B' }}>
            {language === 'zh' ? '📉 多级移动止损' : '📉 Multi-level Trailing Stop Loss'}
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="p-3 rounded" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
              <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '第 1 档' : 'Tier 1'}
              </div>
              <div className="text-sm font-semibold" style={{ color: '#F59E0B' }}>
                {localConfig.risk_management.trailing_sl1_pct}% → {localConfig.risk_management.trailing_sl1_lock}%
              </div>
              <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '盈利 3% 后锁定 1% 利润' : 'After 3% profit, lock 1%'}
              </div>
            </div>

            <div className="p-3 rounded" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
              <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '第 2 档' : 'Tier 2'}
              </div>
              <div className="text-sm font-semibold" style={{ color: '#F59E0B' }}>
                {localConfig.risk_management.trailing_sl2_pct}% → {localConfig.risk_management.trailing_sl2_lock}%
              </div>
              <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
                {language === 'zh' ? '盈利 5% 后锁定 1.5% 利润' : 'After 5% profit, lock 1.5%'}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
