import { useEffect, useMemo, useState, useCallback, useRef, type FormEvent } from 'react'
import useSWR from 'swr'
import { motion, AnimatePresence } from 'framer-motion'
import { createChart, ColorType, CrosshairMode, CandlestickSeries, createSeriesMarkers, type IChartApi, type ISeriesApi, type CandlestickData, type UTCTimestamp, type SeriesMarker } from 'lightweight-charts'
import {
  Play,
  Pause,
  ChevronRight,
  ChevronLeft,
  Clock,
  TrendingUp,
  TrendingDown,
  Activity,
  Brain,
  Zap,
  AlertTriangle,
  CheckCircle2,
  XCircle,
  RefreshCw,
  Layers,
  Eye,
  CandlestickChart as CandlestickIcon,
} from 'lucide-react'
import {
  ResponsiveContainer,
  ComposedChart,
  Area,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceDot,
  Legend,
} from 'recharts'
import { api } from '../lib/api'
import { useLanguage } from '../contexts/LanguageContext'
import { t } from '../i18n/translations'
import { confirmToast } from '../lib/notify'
import { DecisionCard } from './DecisionCard'
import type {
  BacktestStatusPayload,
  BacktestPositionStatus,
  BacktestEquityPoint,
  BacktestTradeEvent,
  BacktestMetrics,
  BacktestKlinesResponse,
  BacktestStartConfig,
  DecisionRecord,
  AIModel,
  Strategy,
  BaselineStrategy,
} from '../types'
import { BacktestRunCard } from './BacktestRunCard'

// ============ Types ============
type WizardStep = 1 | 2 | 3
type ViewTab = 'overview' | 'chart' | 'trades' | 'decisions' | 'baseline_decisions' | 'compare'

const TIMEFRAME_OPTIONS = ['1m', '3m', '5m', '15m', '30m', '1h', '4h', '1d']
const POPULAR_SYMBOLS = ['BTCUSDT', 'ETHUSDT', 'SOLUSDT', 'BNBUSDT', 'XRPUSDT', 'DOGEUSDT']

// ============ Helper Functions ============
const toLocalInput = (date: Date) => {
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000)
  return local.toISOString().slice(0, 16)
}


// ============ Sub Components ============

// Equity Chart Component using Recharts
function BacktestChart({
  equity,
  trades,
  baselineEquity,
}: {
  equity: BacktestEquityPoint[]
  trades: BacktestTradeEvent[]
  baselineEquity?: BacktestEquityPoint[]
}) {
  const chartData = useMemo(() => {
    // Sort baseline data by timestamp for efficient lookup
    const sortedBaseline = baselineEquity?.length
      ? [...baselineEquity].sort((a, b) => a.ts - b.ts)
      : []

    if (sortedBaseline.length) {
      console.log('[BacktestChart] Baseline data received:', sortedBaseline.length, 'points')
    } else {
      console.log('[BacktestChart] No baseline data received')
    }

    // Helper function to find closest baseline point
    const findClosestBaseline = (targetTs: number): number | null => {
      if (!sortedBaseline.length) return null

      // Binary search for closest timestamp
      let left = 0
      let right = sortedBaseline.length - 1
      let closest = sortedBaseline[0]
      let minDiff = Math.abs(sortedBaseline[0].ts - targetTs)

      while (left <= right) {
        const mid = Math.floor((left + right) / 2)
        const diff = Math.abs(sortedBaseline[mid].ts - targetTs)

        if (diff < minDiff) {
          minDiff = diff
          closest = sortedBaseline[mid]
        }

        if (sortedBaseline[mid].ts < targetTs) {
          left = mid + 1
        } else if (sortedBaseline[mid].ts > targetTs) {
          right = mid - 1
        } else {
          return sortedBaseline[mid].equity
        }
      }

      // Only return if within 1 hour (3600000ms) tolerance
      return minDiff < 3600000 ? closest.equity : null
    }

    const data = equity.map((point) => ({
      time: new Date(point.ts).toLocaleString(),
      ts: point.ts,
      equity: point.equity,
      pnl_pct: point.pnl_pct,
      baseline: findClosestBaseline(point.ts),
    }))

    const baselineCount = data.filter(d => d.baseline !== null).length
    console.log('[BacktestChart] Chart data points with baseline:', baselineCount, '/', data.length)

    return data
  }, [equity, baselineEquity])

  // Find trade points to mark on chart
  const tradeMarkers = useMemo(() => {
    if (!trades.length || !equity.length) return []
    return trades
      .filter((t) => t.action.includes('open') || t.action.includes('close'))
      .map((trade) => {
        // Find closest equity point
        const closest = equity.reduce((prev, curr) =>
          Math.abs(curr.ts - trade.ts) < Math.abs(prev.ts - trade.ts) ? curr : prev
        )
        return {
          ts: closest.ts,
          equity: closest.equity,
          action: trade.action,
          symbol: trade.symbol,
          isOpen: trade.action.includes('open'),
        }
      })
      .slice(-30) // Limit markers
  }, [trades, equity])

  // Debug: Check if baseline should be rendered
  console.log('[BacktestChart] Render check - baselineEquity:', baselineEquity?.length, 'chartData baseline count:', chartData.filter(d => d.baseline !== null).length)
  console.log('[BacktestChart] Sample chartData (first 2):', chartData.slice(0, 2))

  return (
    <div className="w-full h-[300px]">
      <ResponsiveContainer width="100%" height="100%">
        <ComposedChart data={chartData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
          <defs>
            <linearGradient id="equityGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#F0B90B" stopOpacity={0.4} />
              <stop offset="95%" stopColor="#F0B90B" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid stroke="rgba(43, 49, 57, 0.5)" strokeDasharray="3 3" />
          <XAxis
            dataKey="time"
            tick={{ fill: '#848E9C', fontSize: 10 }}
            axisLine={{ stroke: '#2B3139' }}
            tickLine={{ stroke: '#2B3139' }}
            hide
          />
          <YAxis
            tick={{ fill: '#848E9C', fontSize: 10 }}
            axisLine={{ stroke: '#2B3139' }}
            tickLine={{ stroke: '#2B3139' }}
            width={60}
            domain={['auto', 'auto']}
          />
          <Tooltip
            contentStyle={{
              background: '#1E2329',
              border: '1px solid #2B3139',
              borderRadius: 8,
              color: '#EAECEF',
            }}
            labelStyle={{ color: '#848E9C' }}
            formatter={(value: number, name: string) => {
              const label = name === 'equity' ? 'AI' : 'Baseline'
              return [`$${value?.toFixed(2) ?? '-'}`, label]
            }}
          />
          {baselineEquity && baselineEquity.length > 0 && (
            <Legend
              wrapperStyle={{ paddingTop: 10 }}
              formatter={(value) => (
                <span style={{ color: '#848E9C', fontSize: 11 }}>
                  {value === 'equity' ? 'AI Strategy' : 'Baseline (Indicators)'}
                </span>
              )}
            />
          )}
          <Area
            type="monotone"
            dataKey="equity"
            name="equity"
            stroke="#F0B90B"
            strokeWidth={2}
            fill="url(#equityGradient)"
            dot={false}
            activeDot={{ r: 4, fill: '#F0B90B' }}
          />
          {/* Baseline curve (solid line) - always render for proper Recharts updates */}
          <Line
            type="monotone"
            dataKey="baseline"
            name="baseline"
            stroke="#00D9FF"
            strokeWidth={2}
            dot={false}
            activeDot={{ r: 3, fill: '#00D9FF' }}
            connectNulls
          />
          {/* Trade markers */}
          {tradeMarkers.map((marker, idx) => (
            <ReferenceDot
              key={`${marker.ts}-${idx}`}
              x={chartData.findIndex((d) => d.ts === marker.ts)}
              y={marker.equity}
              r={4}
              fill={marker.isOpen ? '#0ECB81' : '#F6465D'}
              stroke={marker.isOpen ? '#0ECB81' : '#F6465D'}
            />
          ))}
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  )
}

// Candlestick Chart Component with trade markers
function CandlestickChartComponent({
  runId,
  trades,
  language,
}: {
  runId: string
  trades: BacktestTradeEvent[]
  language: string
}) {
  const chartContainerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const candleSeriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null)

  // Get unique symbols from trades
  const symbols = useMemo(() => {
    const symbolSet = new Set(trades.map((t) => t.symbol))
    return Array.from(symbolSet).sort()
  }, [trades])

  const [selectedSymbol, setSelectedSymbol] = useState<string>(symbols[0] || '')
  const [selectedTimeframe, setSelectedTimeframe] = useState<string>('15m')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const CHART_TIMEFRAMES = ['1m', '3m', '5m', '15m', '30m', '1h', '4h', '1d']

  // Update selected symbol when symbols change
  useEffect(() => {
    if (symbols.length > 0 && !symbols.includes(selectedSymbol)) {
      setSelectedSymbol(symbols[0])
    }
  }, [symbols, selectedSymbol])

  // Filter trades for selected symbol
  const symbolTrades = useMemo(() => {
    return trades.filter((t) => t.symbol === selectedSymbol)
  }, [trades, selectedSymbol])

  // Fetch klines and render chart
  useEffect(() => {
    if (!chartContainerRef.current || !selectedSymbol || !runId) return

    const container = chartContainerRef.current

    // Create chart
    const chart = createChart(container, {
      layout: {
        background: { type: ColorType.Solid, color: '#0B0E11' },
        textColor: '#848E9C',
      },
      grid: {
        vertLines: { color: 'rgba(43, 49, 57, 0.5)' },
        horzLines: { color: 'rgba(43, 49, 57, 0.5)' },
      },
      crosshair: {
        mode: CrosshairMode.Normal,
      },
      rightPriceScale: {
        borderColor: '#2B3139',
      },
      timeScale: {
        borderColor: '#2B3139',
        timeVisible: true,
        secondsVisible: false,
      },
      width: container.clientWidth,
      height: 400,
    })

    chartRef.current = chart

    // Add candlestick series
    const candleSeries = chart.addSeries(CandlestickSeries, {
      upColor: '#0ECB81',
      downColor: '#F6465D',
      borderUpColor: '#0ECB81',
      borderDownColor: '#F6465D',
      wickUpColor: '#0ECB81',
      wickDownColor: '#F6465D',
    })
    candleSeriesRef.current = candleSeries

    // Fetch klines
    setIsLoading(true)
    setError(null)

    api
      .getBacktestKlines(runId, selectedSymbol, selectedTimeframe)
      .then((data: BacktestKlinesResponse) => {
        const klineData: CandlestickData<UTCTimestamp>[] = data.klines.map((k) => ({
          time: k.time as UTCTimestamp,
          open: k.open,
          high: k.high,
          low: k.low,
          close: k.close,
        }))
        candleSeries.setData(klineData)

        // Add trade markers with improved styling
        const markers: SeriesMarker<UTCTimestamp>[] = symbolTrades
          .map((trade) => {
            const tradeTime = Math.floor(trade.ts / 1000)
            // Find closest kline time
            const closestKline = data.klines.reduce((prev, curr) =>
              Math.abs(curr.time - tradeTime) < Math.abs(prev.time - tradeTime) ? curr : prev
            )
            const isOpen = trade.action.includes('open')
            const isLong = trade.side === 'long' || trade.action.includes('long')
            const pnl = trade.realized_pnl

            // Format display text
            let text = ''
            let color = '#0ECB81' // Default green

            if (isOpen) {
              // Opening position: show direction and price
              if (isLong) {
                text = `▲ Long @${trade.price.toFixed(2)}`
                color = '#0ECB81' // Green for long open
              } else {
                text = `▼ Short @${trade.price.toFixed(2)}`
                color = '#F6465D' // Red for short open
              }
            } else {
              // Closing position: show PnL
              const pnlStr = pnl >= 0 ? `+$${pnl.toFixed(2)}` : `-$${Math.abs(pnl).toFixed(2)}`
              text = `✕ ${pnlStr}`
              color = pnl >= 0 ? '#0ECB81' : '#F6465D' // Green for profit, red for loss
            }

            return {
              time: closestKline.time as UTCTimestamp,
              position: isOpen
                ? (isLong ? 'belowBar' as const : 'aboveBar' as const) // Long below, short above
                : (isLong ? 'aboveBar' as const : 'belowBar' as const), // Close opposite
              color,
              shape: 'circle' as const,
              size: 2,
              text,
            }
          })
          .sort((a, b) => (a.time as number) - (b.time as number))

        createSeriesMarkers(candleSeries, markers)
        chart.timeScale().fitContent()
        setIsLoading(false)
      })
      .catch((err) => {
        setError(err.message || 'Failed to load klines')
        setIsLoading(false)
      })

    // Handle resize
    const handleResize = () => {
      if (chartContainerRef.current) {
        chart.applyOptions({ width: chartContainerRef.current.clientWidth })
      }
    }
    window.addEventListener('resize', handleResize)

    return () => {
      window.removeEventListener('resize', handleResize)
      chart.remove()
      chartRef.current = null
      candleSeriesRef.current = null
    }
  }, [runId, selectedSymbol, selectedTimeframe, symbolTrades])

  if (symbols.length === 0) {
    return (
      <div className="py-12 text-center" style={{ color: '#5E6673' }}>
        {language === 'zh' ? '没有交易记录' : 'No trades to display'}
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {/* Symbol and Timeframe selectors */}
      <div className="flex items-center gap-4 flex-wrap">
        <div className="flex items-center gap-2">
          <CandlestickIcon size={16} style={{ color: '#F0B90B' }} />
          <span className="text-sm" style={{ color: '#848E9C' }}>
            {language === 'zh' ? '币种' : 'Symbol'}
          </span>
          <select
            value={selectedSymbol}
            onChange={(e) => setSelectedSymbol(e.target.value)}
            className="px-3 py-1.5 rounded text-sm"
            style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
          >
            {symbols.map((sym) => (
              <option key={sym} value={sym}>
                {sym}
              </option>
            ))}
          </select>
        </div>

        <div className="flex items-center gap-2">
          <Clock size={14} style={{ color: '#848E9C' }} />
          <span className="text-sm" style={{ color: '#848E9C' }}>
            {language === 'zh' ? '周期' : 'Interval'}
          </span>
          <div className="flex rounded overflow-hidden" style={{ border: '1px solid #2B3139' }}>
            {CHART_TIMEFRAMES.map((tf) => (
              <button
                key={tf}
                onClick={() => setSelectedTimeframe(tf)}
                className="px-2.5 py-1 text-xs font-medium transition-colors"
                style={{
                  background: selectedTimeframe === tf ? '#F0B90B' : '#1E2329',
                  color: selectedTimeframe === tf ? '#0B0E11' : '#848E9C',
                }}
              >
                {tf}
              </button>
            ))}
          </div>
        </div>

        <span className="text-xs" style={{ color: '#5E6673' }}>
          ({symbolTrades.length} {language === 'zh' ? '笔交易' : 'trades'})
        </span>
      </div>

      {/* Chart container */}
      <div
        ref={chartContainerRef}
        className="w-full rounded-lg overflow-hidden"
        style={{ background: '#0B0E11', minHeight: 400 }}
      >
        {isLoading && (
          <div className="flex items-center justify-center h-[400px]" style={{ color: '#848E9C' }}>
            <RefreshCw className="animate-spin mr-2" size={16} />
            {language === 'zh' ? '加载K线数据...' : 'Loading kline data...'}
          </div>
        )}
        {error && (
          <div className="flex items-center justify-center h-[400px]" style={{ color: '#F6465D' }}>
            <AlertTriangle className="mr-2" size={16} />
            {error}
          </div>
        )}
      </div>

      {/* Legend */}
      <div className="flex items-center gap-4 text-xs" style={{ color: '#848E9C' }}>
        <div className="flex items-center gap-1.5">
          <div className="w-2.5 h-2.5 rounded-full" style={{ background: '#0ECB81' }} />
          <span>{language === 'zh' ? '开仓/盈利' : 'Open/Profit'}</span>
        </div>
        <div className="flex items-center gap-1.5">
          <div className="w-2.5 h-2.5 rounded-full" style={{ background: '#F6465D' }} />
          <span>{language === 'zh' ? '亏损平仓' : 'Loss Close'}</span>
        </div>
        <span style={{ color: '#5E6673' }}>|</span>
        <span>▲ Long · ▼ Short · ✕ {language === 'zh' ? '平仓' : 'Close'}</span>
      </div>
    </div>
  )
}

// Trade Timeline Component
function TradeTimeline({ trades }: { trades: BacktestTradeEvent[] }) {
  const recentTrades = useMemo(() => [...trades].slice(-20).reverse(), [trades])

  if (recentTrades.length === 0) {
    return (
      <div className="py-12 text-center" style={{ color: '#5E6673' }}>
        No trades yet
      </div>
    )
  }

  return (
    <div className="space-y-2 max-h-[400px] overflow-y-auto pr-2">
      {recentTrades.map((trade, idx) => {
        const isOpen = trade.action.includes('open')
        const isLong = trade.action.includes('long')
        const bgColor = isOpen ? 'rgba(14, 203, 129, 0.1)' : 'rgba(246, 70, 93, 0.1)'
        const borderColor = isOpen ? 'rgba(14, 203, 129, 0.3)' : 'rgba(246, 70, 93, 0.3)'
        const iconColor = isOpen ? '#0ECB81' : '#F6465D'

        return (
          <motion.div
            key={`${trade.ts}-${trade.symbol}-${idx}`}
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: idx * 0.05 }}
            className="p-3 rounded-lg flex items-center gap-3"
            style={{ background: bgColor, border: `1px solid ${borderColor}` }}
          >
            <div
              className="w-8 h-8 rounded-full flex items-center justify-center"
              style={{ background: `${iconColor}20` }}
            >
              {isLong ? (
                <TrendingUp className="w-4 h-4" style={{ color: iconColor }} />
              ) : (
                <TrendingDown className="w-4 h-4" style={{ color: iconColor }} />
              )}
            </div>
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="font-mono font-bold text-sm" style={{ color: '#EAECEF' }}>
                  {trade.symbol.replace('USDT', '')}
                </span>
                <span
                  className="px-2 py-0.5 rounded text-xs font-medium"
                  style={{ background: `${iconColor}20`, color: iconColor }}
                >
                  {trade.action.replace('_', ' ').toUpperCase()}
                </span>
                {trade.leverage && (
                  <span className="text-xs" style={{ color: '#848E9C' }}>
                    {trade.leverage}x
                  </span>
                )}
              </div>
              <div className="text-xs mt-1" style={{ color: '#848E9C' }}>
                {new Date(trade.ts).toLocaleString()} · Qty: {trade.qty.toFixed(4)} · ${trade.price.toFixed(2)}
              </div>
            </div>
            <div className="text-right">
              <div
                className="font-mono font-bold"
                style={{ color: trade.realized_pnl >= 0 ? '#0ECB81' : '#F6465D' }}
              >
                {trade.realized_pnl >= 0 ? '+' : ''}
                {trade.realized_pnl.toFixed(2)}
              </div>
              <div className="text-xs" style={{ color: '#848E9C' }}>
                USDT
              </div>
            </div>
          </motion.div>
        )
      })}
    </div>
  )
}

// Real-time Positions Display Component
function PositionsDisplay({
  positions,
  baselinePositions,
  language,
}: {
  positions: BacktestPositionStatus[]
  baselinePositions?: any[]
  language: string
}) {
  const hasAIPositions = positions && positions.length > 0
  const hasBaselinePositions = baselinePositions && baselinePositions.length > 0

  if (!hasAIPositions && !hasBaselinePositions) {
    return null
  }

  const totalUnrealizedPnL = positions?.reduce((sum, p) => sum + p.unrealized_pnl, 0) || 0
  const totalMargin = positions?.reduce((sum, p) => sum + p.margin_used, 0) || 0

  return (
    <div
      className="mt-3 p-3 rounded-lg"
      style={{ background: 'rgba(30, 35, 41, 0.8)', border: '1px solid #2B3139' }}
    >
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <Activity className="w-4 h-4" style={{ color: '#F0B90B' }} />
          <span className="text-sm font-medium" style={{ color: '#EAECEF' }}>
            {language === 'zh' ? '当前持仓' : 'Active Positions'}
          </span>
          <span
            className="px-1.5 py-0.5 rounded text-xs"
            style={{ background: '#F0B90B20', color: '#F0B90B' }}
          >
            {(positions?.length || 0) + (baselinePositions?.length || 0)}
          </span>
        </div>
        <div className="flex items-center gap-3 text-xs">
          <span style={{ color: '#848E9C' }}>
            {language === 'zh' ? '保证金' : 'Margin'}: ${totalMargin.toFixed(2)}
          </span>
          <span
            className="font-medium"
            style={{ color: totalUnrealizedPnL >= 0 ? '#0ECB81' : '#F6465D' }}
          >
            {language === 'zh' ? '浮盈' : 'Unrealized'}: {totalUnrealizedPnL >= 0 ? '+' : ''}
            ${totalUnrealizedPnL.toFixed(2)}
          </span>
        </div>
      </div>

      <div className="space-y-1.5">
        {/* AI 持仓 */}
        {positions.map((pos) => {
          const isLong = pos.side === 'long'
          const pnlColor = pos.unrealized_pnl >= 0 ? '#0ECB81' : '#F6465D'

          return (
            <motion.div
              key={`ai-${pos.symbol}-${pos.side}`}
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              className="flex items-center justify-between p-2 rounded"
              style={{ background: '#1E2329' }}
            >
              <div className="flex items-center gap-2">
                <div
                  className="w-6 h-6 rounded flex items-center justify-center"
                  style={{ background: isLong ? '#0ECB8120' : '#F6465D20' }}
                >
                  {isLong ? (
                    <TrendingUp className="w-3.5 h-3.5" style={{ color: '#0ECB81' }} />
                  ) : (
                    <TrendingDown className="w-3.5 h-3.5" style={{ color: '#F6465D' }} />
                  )}
                </div>
                <div>
                  <div className="flex items-center gap-1.5">
                    <span className="font-mono font-bold text-sm" style={{ color: '#EAECEF' }}>
                      {pos.symbol.replace('USDT', '')}
                    </span>
                    <span
                      className="px-1 py-0.5 rounded text-[10px] font-medium"
                      style={{
                        background: isLong ? '#0ECB8120' : '#F6465D20',
                        color: isLong ? '#0ECB81' : '#F6465D',
                      }}
                    >
                      {isLong ? 'LONG' : 'SHORT'} {pos.leverage}x
                    </span>
                    <span
                      className="px-1 py-0.5 rounded text-[10px] font-medium"
                      style={{ background: '#3B82F620', color: '#3B82F6' }}
                    >
                      AI
                    </span>
                  </div>
                  <div className="text-[10px]" style={{ color: '#5E6673' }}>
                    {language === 'zh' ? '数量' : 'Qty'}: {pos.quantity.toFixed(4)} ·{' '}
                    {language === 'zh' ? '保证金' : 'Margin'}: ${pos.margin_used.toFixed(2)}
                  </div>
                </div>
              </div>

              <div className="text-right">
                <div className="flex items-center gap-2 text-xs">
                  <span style={{ color: '#848E9C' }}>
                    {language === 'zh' ? '开仓' : 'Entry'}: ${pos.entry_price.toFixed(2)}
                  </span>
                  <span style={{ color: '#EAECEF' }}>
                    {language === 'zh' ? '现价' : 'Mark'}: ${pos.mark_price.toFixed(2)}
                  </span>
                </div>
                <div className="flex items-center justify-end gap-1.5 mt-0.5">
                  <span className="font-mono font-bold" style={{ color: pnlColor }}>
                    {pos.unrealized_pnl >= 0 ? '+' : ''}${pos.unrealized_pnl.toFixed(2)}
                  </span>
                  <span
                    className="px-1 py-0.5 rounded text-[10px] font-medium"
                    style={{ background: `${pnlColor}20`, color: pnlColor }}
                  >
                    {pos.unrealized_pnl_pct >= 0 ? '+' : ''}{pos.unrealized_pnl_pct.toFixed(2)}%
                  </span>
                </div>
              </div>
            </motion.div>
          )
        })}

        {/* Baseline 持仓 */}
        {baselinePositions?.map((pos) => {
          const isLong = pos.side === 'long'
          // Use unrealized_pnl from database (fallback to pnl for compatibility)
          const unrealizedPnl = pos.unrealized_pnl ?? pos.pnl ?? 0
          const pnlColor = (pos.pnl_pct || 0) >= 0 ? '#0ECB81' : '#F6465D'

          return (
            <motion.div
              key={`baseline-${pos.symbol}-${pos.side}`}
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              className="flex items-center justify-between p-2 rounded"
              style={{ background: '#1E2329' }}
            >
              <div className="flex items-center gap-2">
                <div
                  className="w-6 h-6 rounded flex items-center justify-center"
                  style={{ background: isLong ? '#0ECB8120' : '#F6465D20' }}
                >
                  {isLong ? (
                    <TrendingUp className="w-3.5 h-3.5" style={{ color: '#0ECB81' }} />
                  ) : (
                    <TrendingDown className="w-3.5 h-3.5" style={{ color: '#F6465D' }} />
                  )}
                </div>
                <div>
                  <div className="flex items-center gap-1.5">
                    <span className="font-mono font-bold text-sm" style={{ color: '#EAECEF' }}>
                      {pos.symbol.replace('USDT', '')}
                    </span>
                    <span
                      className="px-1 py-0.5 rounded text-[10px] font-medium"
                      style={{
                        background: isLong ? '#0ECB8120' : '#F6465D20',
                        color: isLong ? '#0ECB81' : '#F6465D',
                      }}
                    >
                      {isLong ? 'LONG' : 'SHORT'}
                    </span>
                    <span
                      className="px-1 py-0.5 rounded text-[10px] font-medium"
                      style={{ background: '#FF6B3520', color: '#FF6B35' }}
                    >
                      Baseline
                    </span>
                  </div>
                  <div className="text-[10px]" style={{ color: '#5E6673' }}>
                    {language === 'zh' ? '数量' : 'Qty'}: {pos.quantity.toFixed(4)} ·{' '}
                    {language === 'zh' ? '价值' : 'Value'}: ${pos.value.toFixed(2)}
                  </div>
                </div>
              </div>

              <div className="text-right">
                <div className="flex items-center gap-2 text-xs">
                  <span style={{ color: '#848E9C' }}>
                    {language === 'zh' ? '开仓' : 'Entry'}: ${pos.entry_price.toFixed(2)}
                  </span>
                </div>
                <div className="flex items-center justify-end gap-1.5 mt-0.5">
                  <span className="font-mono font-bold" style={{ color: pnlColor }}>
                    {unrealizedPnl >= 0 ? '+' : ''}${unrealizedPnl.toFixed(2)}
                  </span>
                  <span
                    className="px-1 py-0.5 rounded text-[10px] font-medium"
                    style={{ background: `${pnlColor}20`, color: pnlColor }}
                  >
                    {(pos.pnl_pct || 0) >= 0 ? '+' : ''}{(pos.pnl_pct || 0).toFixed(2)}%
                  </span>
                </div>
              </div>
            </motion.div>
          )
        })}
      </div>
    </div>
  )
}

// ============ Main Component ============
export function BacktestPage() {
  const { language } = useLanguage()
  const tr = useCallback(
    (key: string, params?: Record<string, string | number>) => t(`backtestPage.${key}`, language, params),
    [language]
  )

  // State
  const now = new Date()
  const [wizardStep, setWizardStep] = useState<WizardStep>(1)
  const [viewTab, setViewTab] = useState<ViewTab>('overview')
  const [selectedRunId, setSelectedRunId] = useState<string>()
  const [compareRunIds, setCompareRunIds] = useState<string[]>([])
  const [isStarting, setIsStarting] = useState(false)
  const [toast, setToast] = useState<{ text: string; tone: 'info' | 'error' | 'success' } | null>(null)

  // Filter state
  const [filterStrategy, setFilterStrategy] = useState<string>('')
  const [filterDateRange, setFilterDateRange] = useState<string>('all') // all, today, 3d, 7d, 30d

  // Form state
  const [formState, setFormState] = useState({
    runId: '',
    symbols: 'BTCUSDT,ETHUSDT,SOLUSDT',
    timeframes: ['3m', '15m', '4h'],
    decisionTf: '3m',
    cadence: 20,
    start: toLocalInput(new Date(now.getTime() - 3 * 24 * 3600 * 1000)),
    end: toLocalInput(now),
    balance: 1000,
    fee: 5,
    slippage: 2,
    btcEthLeverage: 5,
    altcoinLeverage: 5,
    fill: 'next_open',
    prompt: 'baseline',
    promptTemplate: 'default',
    customPrompt: '',
    overridePrompt: false,
    cacheAI: true,
    replayOnly: false,
    aiModelId: '',
    strategyId: '', // Optional: use saved strategy from Strategy Studio
    enableBaseline: false, // Enable traditional indicator baseline for comparison
    baselineStrategyId: '', // ID of baseline strategy to use
  })

  // Data fetching
  const { data: runsResp, mutate: refreshRuns } = useSWR(['backtest-runs'], () =>
    api.getBacktestRuns({ limit: 100, offset: 0 })
  , { refreshInterval: 5000 })
  const runs = runsResp?.items ?? []

  const { data: aiModels } = useSWR<AIModel[]>('ai-models', api.getModelConfigs, { refreshInterval: 30000 })
  const { data: strategies } = useSWR<Strategy[]>('strategies', api.getStrategies, { refreshInterval: 30000 })
  const { data: baselineStrategies } = useSWR<BaselineStrategy[]>('baseline-strategies', api.listBaselineStrategies, { refreshInterval: 30000 })

  // Check if selected run is active (running/paused)
  const selectedRunState = runs.find(r => r.run_id === selectedRunId)?.state
  const isActiveRun = selectedRunState === 'running' || selectedRunState === 'paused'

  const { data: status } = useSWR<BacktestStatusPayload>(
    selectedRunId ? ['bt-status', selectedRunId] : null,
    () => api.getBacktestStatus(selectedRunId!),
    { refreshInterval: isActiveRun ? 2000 : 0, dedupingInterval: 1000 }
  )

  const { data: equity } = useSWR<BacktestEquityPoint[]>(
    selectedRunId ? ['bt-equity', selectedRunId] : null,
    () => api.getBacktestEquity(selectedRunId!, '1m', 2000),
    { refreshInterval: isActiveRun ? 5000 : 0, dedupingInterval: 2000 }
  )

  const { data: trades } = useSWR<BacktestTradeEvent[]>(
    selectedRunId ? ['bt-trades', selectedRunId] : null,
    () => api.getBacktestTrades(selectedRunId!, 500),
    { refreshInterval: isActiveRun ? 5000 : 0, dedupingInterval: 2000 }
  )

  const { data: metrics } = useSWR<BacktestMetrics>(
    selectedRunId ? ['bt-metrics', selectedRunId] : null,
    () => api.getBacktestMetrics(selectedRunId!),
    { refreshInterval: isActiveRun ? 10000 : 0, dedupingInterval: 5000 }
  )

  const { data: decisions } = useSWR<DecisionRecord[]>(
    selectedRunId ? ['bt-decisions', selectedRunId] : null,
    () => api.getBacktestDecisions(selectedRunId!, 30),
    { refreshInterval: isActiveRun ? 5000 : 0, dedupingInterval: 2000 }
  )

  const { data: baselineDecisions } = useSWR<any[]>(
    selectedRunId ? ['bt-baseline-decisions', selectedRunId] : null,
    () => api.getBaselineDecisions(selectedRunId!),
    { refreshInterval: isActiveRun ? 5000 : 0, dedupingInterval: 2000 }
  )

  const { data: backtestConfig } = useSWR<BacktestStartConfig>(
    selectedRunId ? ['bt-config', selectedRunId] : null,
    () => api.getBacktestConfig(selectedRunId!),
    { revalidateOnFocus: false, dedupingInterval: 10000 }
  )

  // Baseline data - fetch unconditionally to avoid async timing issues
  // Will be filtered in rendering based on backtestConfig.enable_baseline
  const { data: baselineEquity } = useSWR<BacktestEquityPoint[]>(
    selectedRunId ? ['bt-baseline-equity', selectedRunId] : null,
    () => api.getBaselineEquity(selectedRunId!),
    { refreshInterval: isActiveRun ? 5000 : 0, dedupingInterval: 2000 }
  )

  // Debug: Log baseline data fetch result
  useEffect(() => {
    console.log('[BacktestPage] baselineEquity data:', baselineEquity?.length, 'points')
    if (baselineEquity?.length) {
      console.log('[BacktestPage] First baseline point:', baselineEquity[0])
    }
  }, [baselineEquity])

  const selectedRun = runs.find((r) => r.run_id === selectedRunId)
  const selectedModel = aiModels?.find((m) => m.id === formState.aiModelId)
  const selectedStrategy = strategies?.find((s) => s.id === formState.strategyId)

  // Filter runs based on strategy and date range
  // Get unique strategy names from strategies list
  const uniqueStrategyLabels = useMemo(() => {
    if (!strategies) return []
    return strategies.map(s => s.name).filter(Boolean).sort()
  }, [strategies])

  const filteredRuns = useMemo(() => {
    let result = [...runs]

    // Filter by strategy label (exact match)
    if (filterStrategy) {
      result = result.filter(r => r.label === filterStrategy)
    }

    // Filter by date range
    if (filterDateRange !== 'all') {
      const now = new Date()
      result = result.filter(r => {
        // Parse run_id date: bt_20251231_170407 or evo-20251229-0812
        const match = r.run_id.match(/(\d{4})(\d{2})(\d{2})[_-](\d{2})(\d{2})(\d{2})?/)
        if (!match) return true

        const runDate = new Date(`${match[1]}-${match[2]}-${match[3]}T${match[4]}:${match[5]}:${match[6] || '00'}`)

        if (filterDateRange === 'today') {
          // Same calendar day
          return runDate.toDateString() === now.toDateString()
        }

        const ranges: Record<string, number> = {
          '3d': 3 * 24 * 60 * 60 * 1000,
          '7d': 7 * 24 * 60 * 60 * 1000,
          '30d': 30 * 24 * 60 * 60 * 1000,
        }
        const cutoff = now.getTime() - (ranges[filterDateRange] || 0)
        return runDate.getTime() >= cutoff
      })
    }

    return result
  }, [runs, filterStrategy, filterDateRange])

  // Check if selected strategy has dynamic coin source
  const strategyHasDynamicCoins = useMemo(() => {
    if (!selectedStrategy) return false
    const coinSource = selectedStrategy.config?.coin_source
    if (!coinSource) return false

    // Check explicit source_type
    if (coinSource.source_type === 'coinpool' || coinSource.source_type === 'oi_top') {
      return true
    }
    if (coinSource.source_type === 'mixed' && (coinSource.use_coin_pool || coinSource.use_oi_top)) {
      return true
    }

    // Also check flags for backward compatibility (when source_type is empty or not set)
    const srcType = coinSource.source_type as string
    if (!srcType) {
      if (coinSource.use_coin_pool || coinSource.use_oi_top) {
        return true
      }
    }

    return false
  }, [selectedStrategy])

  // Get coin source description
  const coinSourceDescription = useMemo(() => {
    if (!selectedStrategy?.config?.coin_source) return null
    const cs = selectedStrategy.config.coin_source

    // Infer source_type from flags if empty (backward compatibility)
    let sourceType = cs.source_type as string
    if (!sourceType) {
      if (cs.use_coin_pool && cs.use_oi_top) {
        sourceType = 'mixed'
      } else if (cs.use_coin_pool) {
        sourceType = 'coinpool'
      } else if (cs.use_oi_top) {
        sourceType = 'oi_top'
      } else if (cs.static_coins?.length) {
        sourceType = 'static'
      }
    }

    switch (sourceType) {
      case 'coinpool':
        return { type: 'AI500', limit: cs.coin_pool_limit || 30 }
      case 'oi_top':
        return { type: 'OI Top', limit: cs.oi_top_limit || 30 }
      case 'mixed':
        const sources = []
        if (cs.use_coin_pool) sources.push(`AI500(${cs.coin_pool_limit || 30})`)
        if (cs.use_oi_top) sources.push(`OI Top(${cs.oi_top_limit || 30})`)
        if (cs.static_coins?.length) sources.push(`Static(${cs.static_coins.length})`)
        return { type: 'Mixed', desc: sources.join(' + ') }
      case 'static':
        return { type: 'Static', coins: cs.static_coins || [] }
      default:
        return null
    }
  }, [selectedStrategy])

  // Auto-select first model
  useEffect(() => {
    if (!formState.aiModelId && aiModels?.length) {
      const enabled = aiModels.find((m) => m.enabled)
      if (enabled) setFormState((s) => ({ ...s, aiModelId: enabled.id }))
    }
  }, [aiModels, formState.aiModelId])

  // Auto-select first run
  useEffect(() => {
    if (!selectedRunId && runs.length > 0) {
      setSelectedRunId(runs[0].run_id)
    }
  }, [runs, selectedRunId])

  // Handlers
  const handleFormChange = (key: string, value: string | number | boolean | string[]) => {
    setFormState((prev) => ({ ...prev, [key]: value }))
  }

  const handleStart = async (event: FormEvent) => {
    event.preventDefault()
    if (!selectedModel?.enabled) {
      setToast({ text: tr('toasts.selectModel'), tone: 'error' })
      return
    }

    try {
      setIsStarting(true)
      const start = new Date(formState.start).getTime()
      const end = new Date(formState.end).getTime()
      if (end <= start) throw new Error(tr('toasts.invalidRange'))

      // Parse user symbols - if using dynamic coin strategy, allow empty
      const userSymbols = formState.symbols.split(',').map((s) => s.trim()).filter(Boolean)

      // Only send empty symbols if user deliberately cleared them and strategy has dynamic coin source
      const symbolsToSend = (userSymbols.length === 0 && strategyHasDynamicCoins) ? [] : userSymbols

      const payload = await api.startBacktest({
        run_id: formState.runId.trim() || undefined,
        strategy_id: formState.strategyId || undefined, // Use saved strategy from Strategy Studio
        symbols: symbolsToSend,
        timeframes: formState.timeframes,
        decision_timeframe: formState.decisionTf,
        decision_cadence_nbars: formState.cadence,
        start_ts: Math.floor(start / 1000),
        end_ts: Math.floor(end / 1000),
        initial_balance: formState.balance,
        fee_bps: formState.fee,
        slippage_bps: formState.slippage,
        fill_policy: formState.fill,
        prompt_variant: formState.prompt,
        prompt_template: formState.promptTemplate,
        custom_prompt: formState.customPrompt.trim() || undefined,
        override_prompt: formState.overridePrompt,
        cache_ai: formState.cacheAI,
        replay_only: formState.replayOnly,
        ai_model_id: formState.aiModelId,
        leverage: {
          btc_eth_leverage: formState.btcEthLeverage,
          altcoin_leverage: formState.altcoinLeverage,
        },
        enable_baseline: formState.enableBaseline,
        baseline_strategy_id: formState.baselineStrategyId,
      })

      setToast({ text: tr('toasts.startSuccess', { id: payload.run_id }), tone: 'success' })
      setSelectedRunId(payload.run_id)
      setWizardStep(1)
      await refreshRuns()
    } catch (error: unknown) {
      const errMsg = error instanceof Error ? error.message : tr('toasts.startFailed')
      setToast({ text: errMsg, tone: 'error' })
    } finally {
      setIsStarting(false)
    }
  }

  const handleControl = async (action: 'pause' | 'resume' | 'stop') => {
    if (!selectedRunId) return
    try {
      if (action === 'pause') await api.pauseBacktest(selectedRunId)
      if (action === 'resume') await api.resumeBacktest(selectedRunId)
      if (action === 'stop') await api.stopBacktest(selectedRunId)
      setToast({ text: tr('toasts.actionSuccess', { action, id: selectedRunId }), tone: 'success' })
      await refreshRuns()
    } catch (error: unknown) {
      const errMsg = error instanceof Error ? error.message : tr('toasts.actionFailed')
      setToast({ text: errMsg, tone: 'error' })
    }
  }

  const handleForceResume = async () => {
    if (!selectedRunId) return
    try {
      await api.forceResumeBacktest(selectedRunId)
      setToast({ text: tr('toasts.actionSuccess', { action: 'force-resume', id: selectedRunId }), tone: 'success' })
      await refreshRuns()
    } catch (error: unknown) {
      const errMsg = error instanceof Error ? error.message : tr('toasts.actionFailed')
      setToast({ text: errMsg, tone: 'error' })
    }
  }

  const handleDelete = async (runId?: string) => {
    const targetRunId = runId || selectedRunId
    if (!targetRunId) return
    const confirmed = await confirmToast(tr('toasts.confirmDelete', { id: targetRunId }), {
      title: language === 'zh' ? '确认删除' : 'Confirm Delete',
      okText: language === 'zh' ? '删除' : 'Delete',
      cancelText: language === 'zh' ? '取消' : 'Cancel',
    })
    if (!confirmed) return
    try {
      await api.deleteBacktestRun(targetRunId)
      setToast({ text: tr('toasts.deleteSuccess'), tone: 'success' })
      if (selectedRunId === targetRunId) {
        setSelectedRunId(undefined)
      }
      await refreshRuns()
    } catch (error: unknown) {
      const errMsg = error instanceof Error ? error.message : tr('toasts.deleteFailed')
      setToast({ text: errMsg, tone: 'error' })
    }
  }

  const handleExport = async () => {
    if (!selectedRunId) return
    try {
      const blob = await api.exportBacktest(selectedRunId)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `${selectedRunId}_export.zip`
      link.click()
      URL.revokeObjectURL(url)
      setToast({ text: tr('toasts.exportSuccess', { id: selectedRunId }), tone: 'success' })
    } catch (error: unknown) {
      const errMsg = error instanceof Error ? error.message : tr('toasts.exportFailed')
      setToast({ text: errMsg, tone: 'error' })
    }
  }

  const toggleCompare = (runId: string) => {
    setCompareRunIds((prev) =>
      prev.includes(runId) ? prev.filter((id) => id !== runId) : [...prev, runId].slice(-3)
    )
  }

  const quickRanges = [
    { label: language === 'zh' ? '24小时' : '24h', hours: 24 },
    { label: language === 'zh' ? '3天' : '3d', hours: 72 },
    { label: language === 'zh' ? '7天' : '7d', hours: 168 },
    { label: language === 'zh' ? '30天' : '30d', hours: 720 },
  ]

  const applyQuickRange = (hours: number) => {
    const endDate = new Date()
    const startDate = new Date(endDate.getTime() - hours * 3600 * 1000)
    handleFormChange('start', toLocalInput(startDate))
    handleFormChange('end', toLocalInput(endDate))
  }

  const getStateColor = (state: string) => {
    switch (state) {
      case 'running':
        return '#F0B90B'
      case 'completed':
        return '#0ECB81'
      case 'failed':
      case 'liquidated':
        return '#F6465D'
      case 'paused':
        return '#848E9C'
      default:
        return '#848E9C'
    }
  }

  const getStateIcon = (state: string) => {
    switch (state) {
      case 'running':
        return <Activity className="w-4 h-4" />
      case 'completed':
        return <CheckCircle2 className="w-4 h-4" />
      case 'failed':
      case 'liquidated':
        return <XCircle className="w-4 h-4" />
      case 'paused':
        return <Pause className="w-4 h-4" />
      default:
        return <Clock className="w-4 h-4" />
    }
  }

  // Render
  return (
    <div className="space-y-6">
      {/* Toast */}
      <AnimatePresence>
        {toast && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            className="p-3 rounded-lg text-sm"
            style={{
              background:
                toast.tone === 'error'
                  ? 'rgba(246,70,93,0.15)'
                  : toast.tone === 'success'
                    ? 'rgba(14,203,129,0.15)'
                    : 'rgba(240,185,11,0.15)',
              color: toast.tone === 'error' ? '#F6465D' : toast.tone === 'success' ? '#0ECB81' : '#F0B90B',
              border: `1px solid ${toast.tone === 'error' ? 'rgba(246,70,93,0.3)' : toast.tone === 'success' ? 'rgba(14,203,129,0.3)' : 'rgba(240,185,11,0.3)'}`,
            }}
          >
            {toast.text}
          </motion.div>
        )}
      </AnimatePresence>

      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-3" style={{ color: '#EAECEF' }}>
            <Brain className="w-7 h-7" style={{ color: '#F0B90B' }} />
            {tr('title')}
          </h1>
          <p className="text-sm mt-1" style={{ color: '#848E9C' }}>
            {tr('subtitle')}
          </p>
        </div>
        <button
          onClick={() => setWizardStep(1)}
          className="px-4 py-2 rounded-lg font-medium flex items-center gap-2 transition-all hover:opacity-90"
          style={{ background: '#F0B90B', color: '#0B0E11' }}
        >
          <Play className="w-4 h-4" />
          {language === 'zh' ? '新建回测' : 'New Backtest'}
        </button>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-6">
        {/* Left Panel - Config / History */}
        <div className="space-y-4">
          {/* Wizard */}
          <div className="binance-card p-5">
            <div className="flex items-center gap-2 mb-4">
              {[1, 2, 3].map((step) => (
                <div key={step} className="flex items-center">
                  <button
                    onClick={() => setWizardStep(step as WizardStep)}
                    className="w-8 h-8 rounded-full flex items-center justify-center text-sm font-bold transition-all"
                    style={{
                      background: wizardStep >= step ? '#F0B90B' : '#2B3139',
                      color: wizardStep >= step ? '#0B0E11' : '#848E9C',
                    }}
                  >
                    {step}
                  </button>
                  {step < 3 && (
                    <div
                      className="w-8 h-0.5 mx-1"
                      style={{ background: wizardStep > step ? '#F0B90B' : '#2B3139' }}
                    />
                  )}
                </div>
              ))}
              <span className="ml-2 text-xs" style={{ color: '#848E9C' }}>
                {wizardStep === 1
                  ? language === 'zh'
                    ? '选择模型'
                    : 'Select Model'
                  : wizardStep === 2
                    ? language === 'zh'
                      ? '配置参数'
                      : 'Configure'
                    : language === 'zh'
                      ? '确认启动'
                      : 'Confirm'}
              </span>
            </div>

            <form onSubmit={handleStart}>
              <AnimatePresence mode="wait">
                {/* Step 1: Model & Symbols */}
                {wizardStep === 1 && (
                  <motion.div
                    key="step1"
                    initial={{ opacity: 0, x: 20 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: -20 }}
                    className="space-y-4"
                  >
                    <div>
                      <label className="block text-xs mb-2" style={{ color: '#848E9C' }}>
                        {tr('form.aiModelLabel')}
                      </label>
                      <select
                        className="w-full p-3 rounded-lg text-sm"
                        style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
                        value={formState.aiModelId}
                        onChange={(e) => handleFormChange('aiModelId', e.target.value)}
                      >
                        <option value="">{tr('form.selectAiModel')}</option>
                        {aiModels?.map((m) => (
                          <option key={m.id} value={m.id}>
                            {m.displayName || m.name} ({m.provider}) {!m.enabled && '⚠️'}
                          </option>
                        ))}
                      </select>
                      {selectedModel && (
                        <div className="mt-2 flex items-center gap-2 text-xs">
                          <span
                            className="px-2 py-0.5 rounded"
                            style={{
                              background: selectedModel.enabled ? 'rgba(14,203,129,0.1)' : 'rgba(246,70,93,0.1)',
                              color: selectedModel.enabled ? '#0ECB81' : '#F6465D',
                            }}
                          >
                            {selectedModel.enabled ? tr('form.enabled') : tr('form.disabled')}
                          </span>
                        </div>
                      )}
                    </div>

                    {/* Strategy Selection (Optional) */}
                    <div>
                      <label className="block text-xs mb-2" style={{ color: '#848E9C' }}>
                        {language === 'zh' ? '策略配置（可选）' : 'Strategy (Optional)'}
                      </label>
                      <select
                        className="w-full p-3 rounded-lg text-sm"
                        style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
                        value={formState.strategyId}
                        onChange={(e) => handleFormChange('strategyId', e.target.value)}
                      >
                        <option value="">{language === 'zh' ? '不使用保存的策略' : 'No saved strategy'}</option>
                        {strategies?.map((s) => (
                          <option key={s.id} value={s.id}>
                            {s.name} {s.is_active && '✓'} {s.is_default && '⭐'}
                          </option>
                        ))}
                      </select>
                      {formState.strategyId && coinSourceDescription && (
                        <div className="mt-2 p-2 rounded" style={{ background: 'rgba(240,185,11,0.1)', border: '1px solid rgba(240,185,11,0.2)' }}>
                          <div className="flex items-center gap-2 text-xs">
                            <span style={{ color: '#F0B90B' }}>
                              {language === 'zh' ? '币种来源:' : 'Coin Source:'}
                            </span>
                            <span className="font-medium" style={{ color: '#EAECEF' }}>
                              {coinSourceDescription.type}
                              {coinSourceDescription.limit && ` (${coinSourceDescription.limit})`}
                              {coinSourceDescription.desc && ` - ${coinSourceDescription.desc}`}
                            </span>
                          </div>
                          {strategyHasDynamicCoins && (
                            <div className="text-xs mt-1" style={{ color: '#F0B90B' }}>
                              {language === 'zh'
                                ? '⚡ 清空下方币种输入框即可使用策略的动态币种'
                                : '⚡ Clear the symbols field below to use strategy\'s dynamic coins'}
                            </div>
                          )}
                        </div>
                      )}
                    </div>

                    <div>
                      <label className="block text-xs mb-2" style={{ color: '#848E9C' }}>
                        {tr('form.symbolsLabel')}
                        {strategyHasDynamicCoins && (
                          <span className="ml-2" style={{ color: '#5E6673' }}>
                            ({language === 'zh' ? '可选 - 策略已配置币种来源' : 'Optional - strategy has coin source'})
                          </span>
                        )}
                      </label>
                      {!strategyHasDynamicCoins && (
                        <div className="flex flex-wrap gap-1 mb-2">
                          {POPULAR_SYMBOLS.map((sym) => {
                            const isSelected = formState.symbols.includes(sym)
                            return (
                              <button
                                key={sym}
                                type="button"
                                onClick={() => {
                                  const current = formState.symbols.split(',').map((s) => s.trim()).filter(Boolean)
                                  const updated = isSelected
                                    ? current.filter((s) => s !== sym)
                                    : [...current, sym]
                                  handleFormChange('symbols', updated.join(','))
                                }}
                                className="px-2 py-1 rounded text-xs transition-all"
                                style={{
                                  background: isSelected ? 'rgba(240,185,11,0.15)' : '#1E2329',
                                  border: `1px solid ${isSelected ? '#F0B90B' : '#2B3139'}`,
                                  color: isSelected ? '#F0B90B' : '#848E9C',
                                }}
                              >
                                {sym.replace('USDT', '')}
                              </button>
                            )
                          })}
                        </div>
                      )}
                      <div className="relative">
                        <textarea
                          className="w-full p-2 rounded-lg text-xs font-mono"
                          style={{
                            background: '#0B0E11',
                            border: '1px solid #2B3139',
                            color: '#EAECEF',
                          }}
                          value={formState.symbols}
                          onChange={(e) => handleFormChange('symbols', e.target.value)}
                          rows={2}
                          placeholder={strategyHasDynamicCoins
                            ? (language === 'zh' ? '留空将使用策略配置的币种来源' : 'Leave empty to use strategy coin source')
                            : ''
                          }
                        />
                        {strategyHasDynamicCoins && formState.symbols && (
                          <button
                            type="button"
                            onClick={() => handleFormChange('symbols', '')}
                            className="absolute top-2 right-2 px-2 py-1 rounded text-xs"
                            style={{ background: '#F0B90B', color: '#0B0E11' }}
                          >
                            {language === 'zh' ? '清空使用策略币种' : 'Clear to use strategy'}
                          </button>
                        )}
                      </div>
                    </div>

                    <button
                      type="button"
                      onClick={() => setWizardStep(2)}
                      disabled={!selectedModel?.enabled}
                      className="w-full py-2.5 rounded-lg font-medium flex items-center justify-center gap-2 transition-all disabled:opacity-50"
                      style={{ background: '#F0B90B', color: '#0B0E11' }}
                    >
                      {language === 'zh' ? '下一步' : 'Next'}
                      <ChevronRight className="w-4 h-4" />
                    </button>
                  </motion.div>
                )}

                {/* Step 2: Parameters */}
                {wizardStep === 2 && (
                  <motion.div
                    key="step2"
                    initial={{ opacity: 0, x: 20 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: -20 }}
                    className="space-y-4"
                  >
                    <div>
                      <label className="block text-xs mb-2" style={{ color: '#848E9C' }}>
                        {tr('form.timeRangeLabel')}
                      </label>
                      <div className="flex flex-wrap gap-1 mb-2">
                        {quickRanges.map((r) => (
                          <button
                            key={r.hours}
                            type="button"
                            onClick={() => applyQuickRange(r.hours)}
                            className="px-3 py-1 rounded text-xs"
                            style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
                          >
                            {r.label}
                          </button>
                        ))}
                      </div>
                      <div className="grid grid-cols-2 gap-2">
                        <input
                          type="datetime-local"
                          className="p-2 rounded-lg text-xs"
                          style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
                          value={formState.start}
                          onChange={(e) => handleFormChange('start', e.target.value)}
                        />
                        <input
                          type="datetime-local"
                          className="p-2 rounded-lg text-xs"
                          style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
                          value={formState.end}
                          onChange={(e) => handleFormChange('end', e.target.value)}
                        />
                      </div>
                    </div>

                    <div>
                      <label className="block text-xs mb-2" style={{ color: '#848E9C' }}>
                        {language === 'zh' ? '时间周期' : 'Timeframes'}
                      </label>
                      <div className="flex flex-wrap gap-1">
                        {TIMEFRAME_OPTIONS.map((tf) => {
                          const isSelected = formState.timeframes.includes(tf)
                          return (
                            <button
                              key={tf}
                              type="button"
                              onClick={() => {
                                const updated = isSelected
                                  ? formState.timeframes.filter((t) => t !== tf)
                                  : [...formState.timeframes, tf]
                                if (updated.length > 0) handleFormChange('timeframes', updated)
                              }}
                              className="px-2 py-1 rounded text-xs transition-all"
                              style={{
                                background: isSelected ? 'rgba(240,185,11,0.15)' : '#1E2329',
                                border: `1px solid ${isSelected ? '#F0B90B' : '#2B3139'}`,
                                color: isSelected ? '#F0B90B' : '#848E9C',
                              }}
                            >
                              {tf}
                            </button>
                          )
                        })}
                      </div>
                    </div>

                    <div className="grid grid-cols-2 gap-3">
                      <div>
                        <label className="block text-xs mb-1" style={{ color: '#848E9C' }}>
                          {tr('form.initialBalanceLabel')}
                        </label>
                        <input
                          type="number"
                          className="w-full p-2 rounded-lg text-xs"
                          style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
                          value={formState.balance}
                          onChange={(e) => handleFormChange('balance', Number(e.target.value))}
                        />
                      </div>
                      <div>
                        <label className="block text-xs mb-1" style={{ color: '#848E9C' }}>
                          {tr('form.decisionTfLabel')}
                        </label>
                        <select
                          className="w-full p-2 rounded-lg text-xs"
                          style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
                          value={formState.decisionTf}
                          onChange={(e) => handleFormChange('decisionTf', e.target.value)}
                        >
                          {formState.timeframes.map((tf) => (
                            <option key={tf} value={tf}>
                              {tf}
                            </option>
                          ))}
                        </select>
                      </div>
                    </div>

                    <div className="flex gap-2">
                      <button
                        type="button"
                        onClick={() => setWizardStep(1)}
                        className="flex-1 py-2 rounded-lg font-medium flex items-center justify-center gap-2"
                        style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
                      >
                        <ChevronLeft className="w-4 h-4" />
                        {language === 'zh' ? '上一步' : 'Back'}
                      </button>
                      <button
                        type="button"
                        onClick={() => setWizardStep(3)}
                        className="flex-1 py-2 rounded-lg font-medium flex items-center justify-center gap-2"
                        style={{ background: '#F0B90B', color: '#0B0E11' }}
                      >
                        {language === 'zh' ? '下一步' : 'Next'}
                        <ChevronRight className="w-4 h-4" />
                      </button>
                    </div>
                  </motion.div>
                )}

                {/* Step 3: Advanced & Confirm */}
                {wizardStep === 3 && (
                  <motion.div
                    key="step3"
                    initial={{ opacity: 0, x: 20 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: -20 }}
                    className="space-y-4"
                  >
                    <div className="grid grid-cols-2 gap-3">
                      <div>
                        <label className="block text-xs mb-1" style={{ color: '#848E9C' }}>
                          {tr('form.btcEthLeverageLabel')}
                        </label>
                        <input
                          type="number"
                          className="w-full p-2 rounded-lg text-xs"
                          style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
                          value={formState.btcEthLeverage}
                          onChange={(e) => handleFormChange('btcEthLeverage', Number(e.target.value))}
                        />
                      </div>
                      <div>
                        <label className="block text-xs mb-1" style={{ color: '#848E9C' }}>
                          {tr('form.altcoinLeverageLabel')}
                        </label>
                        <input
                          type="number"
                          className="w-full p-2 rounded-lg text-xs"
                          style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
                          value={formState.altcoinLeverage}
                          onChange={(e) => handleFormChange('altcoinLeverage', Number(e.target.value))}
                        />
                      </div>
                    </div>

                    <div className="grid grid-cols-3 gap-2">
                      <div>
                        <label className="block text-xs mb-1" style={{ color: '#848E9C' }}>
                          {tr('form.feeLabel')}
                        </label>
                        <input
                          type="number"
                          className="w-full p-2 rounded-lg text-xs"
                          style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
                          value={formState.fee}
                          onChange={(e) => handleFormChange('fee', Number(e.target.value))}
                        />
                      </div>
                      <div>
                        <label className="block text-xs mb-1" style={{ color: '#848E9C' }}>
                          {tr('form.slippageLabel')}
                        </label>
                        <input
                          type="number"
                          className="w-full p-2 rounded-lg text-xs"
                          style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
                          value={formState.slippage}
                          onChange={(e) => handleFormChange('slippage', Number(e.target.value))}
                        />
                      </div>
                      <div>
                        <label className="block text-xs mb-1" style={{ color: '#848E9C' }}>
                          {tr('form.cadenceLabel')}
                        </label>
                        <input
                          type="number"
                          className="w-full p-2 rounded-lg text-xs"
                          style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
                          value={formState.cadence}
                          onChange={(e) => handleFormChange('cadence', Number(e.target.value))}
                        />
                      </div>
                    </div>

                    <div>
                      <label className="block text-xs mb-1" style={{ color: '#848E9C' }}>
                        {language === 'zh' ? '策略风格' : 'Strategy Style'}
                      </label>
                      <div className="flex flex-wrap gap-1">
                        {['baseline', 'aggressive', 'conservative', 'scalping'].map((p) => (
                          <button
                            key={p}
                            type="button"
                            onClick={() => handleFormChange('prompt', p)}
                            className="px-3 py-1.5 rounded text-xs transition-all"
                            style={{
                              background: formState.prompt === p ? 'rgba(240,185,11,0.15)' : '#1E2329',
                              border: `1px solid ${formState.prompt === p ? '#F0B90B' : '#2B3139'}`,
                              color: formState.prompt === p ? '#F0B90B' : '#848E9C',
                            }}
                          >
                            {tr(`form.promptPresets.${p}`)}
                          </button>
                        ))}
                      </div>
                    </div>

                    <div className="flex flex-wrap gap-4 text-xs" style={{ color: '#848E9C' }}>
                      <label className="flex items-center gap-2 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={formState.cacheAI}
                          onChange={(e) => handleFormChange('cacheAI', e.target.checked)}
                          className="accent-[#F0B90B]"
                        />
                        {tr('form.cacheAiLabel')}
                      </label>
                      <label className="flex items-center gap-2 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={formState.replayOnly}
                          onChange={(e) => handleFormChange('replayOnly', e.target.checked)}
                          className="accent-[#F0B90B]"
                        />
                        {tr('form.replayOnlyLabel')}
                      </label>
                      <label className="flex items-center gap-2 cursor-pointer" title={language === 'zh' ? '并行运行传统指标策略作为基线对比' : 'Run traditional indicator strategy in parallel as baseline'}>
                        <input
                          type="checkbox"
                          checked={formState.enableBaseline}
                          onChange={(e) => handleFormChange('enableBaseline', e.target.checked)}
                          className="accent-[#F0B90B]"
                        />
                        {language === 'zh' ? '启用基线对比' : 'Enable Baseline'}
                      </label>
                    </div>

                    {/* Baseline Strategy Selector */}
                    {formState.enableBaseline && (
                      <div className="space-y-3 ml-6 p-3 bg-[#0B0E11] rounded">
                        <label className="block">
                          <span className="text-sm text-gray-400">{language === 'zh' ? 'Baseline策略' : 'Baseline Strategy'}</span>
                          <select
                            value={formState.baselineStrategyId}
                            onChange={(e) => handleFormChange('baselineStrategyId', e.target.value)}
                            className="w-full mt-1 px-3 py-2 bg-[#1a1d24] border border-gray-700 rounded text-sm"
                          >
                            <option value="">{language === 'zh' ? '-- 选择baseline策略 --' : '-- Select baseline strategy --'}</option>
                            {baselineStrategies?.map((bs) => (
                              <option key={bs.id} value={bs.id}>
                                {bs.name}
                                {bs.stats && ` (${language === 'zh' ? '平均收益' : 'Avg Return'}: ${bs.stats.avg_return_pct.toFixed(2)}%)`}
                              </option>
                            ))}
                          </select>
                        </label>
                      </div>
                    )}

                    <div className="flex gap-2">
                      <button
                        type="button"
                        onClick={() => setWizardStep(2)}
                        className="flex-1 py-2 rounded-lg font-medium flex items-center justify-center gap-2"
                        style={{ background: '#1E2329', border: '1px solid #2B3139', color: '#EAECEF' }}
                      >
                        <ChevronLeft className="w-4 h-4" />
                        {language === 'zh' ? '上一步' : 'Back'}
                      </button>
                      <button
                        type="submit"
                        disabled={isStarting}
                        className="flex-1 py-2 rounded-lg font-bold flex items-center justify-center gap-2 disabled:opacity-50"
                        style={{ background: '#F0B90B', color: '#0B0E11' }}
                      >
                        {isStarting ? (
                          <RefreshCw className="w-4 h-4 animate-spin" />
                        ) : (
                          <Zap className="w-4 h-4" />
                        )}
                        {isStarting ? tr('starting') : tr('start')}
                      </button>
                    </div>
                  </motion.div>
                )}
              </AnimatePresence>
            </form>
          </div>

          {/* Run History */}
          <div className="binance-card p-4">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-bold flex items-center gap-2" style={{ color: '#EAECEF' }}>
                <Layers className="w-4 h-4" style={{ color: '#F0B90B' }} />
                {tr('runList.title')}
              </h3>
              <span className="text-xs" style={{ color: '#848E9C' }}>
                {filteredRuns.length}/{runs.length} {language === 'zh' ? '条' : 'runs'}
              </span>
            </div>

            {/* Filters */}
            <div className="space-y-2 mb-3">
              {/* Date Range Filter */}
              <div className="flex flex-wrap gap-1">
                {[
                  { key: 'all', label: language === 'zh' ? '全部' : 'All' },
                  { key: 'today', label: language === 'zh' ? '今天' : 'Today' },
                  { key: '3d', label: '3D' },
                  { key: '7d', label: '7D' },
                  { key: '30d', label: '30D' },
                ].map((opt) => (
                  <button
                    key={opt.key}
                    onClick={() => setFilterDateRange(opt.key)}
                    className="px-2 py-1 rounded text-xs transition-all"
                    style={{
                      background: filterDateRange === opt.key ? 'rgba(240,185,11,0.15)' : '#1E2329',
                      border: `1px solid ${filterDateRange === opt.key ? '#F0B90B' : '#2B3139'}`,
                      color: filterDateRange === opt.key ? '#F0B90B' : '#848E9C',
                    }}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>

              {/* Strategy Filter */}
              <select
                value={filterStrategy}
                onChange={(e) => setFilterStrategy(e.target.value)}
                className="w-full px-2 py-1.5 rounded text-xs"
                style={{ background: '#0B0E11', border: '1px solid #2B3139', color: '#EAECEF' }}
              >
                <option value="">{language === 'zh' ? '全部策略' : 'All Strategies'}</option>
                {uniqueStrategyLabels.map((label) => (
                  <option key={label} value={label}>{label}</option>
                ))}
              </select>
            </div>

            <div className="space-y-2 max-h-[400px] overflow-y-auto">
              {filteredRuns.length === 0 ? (
                <div className="py-8 text-center text-sm" style={{ color: '#5E6673' }}>
                  {tr('emptyStates.noRuns')}
                </div>
              ) : (
                [...filteredRuns].sort((a, b) => {
                  // Non-completed states priority: running, paused, stopped, created
                  const nonCompletedStates = ['running', 'paused', 'stopped', 'created']
                  const aIsNonCompleted = nonCompletedStates.includes(a.state)
                  const bIsNonCompleted = nonCompletedStates.includes(b.state)

                  if (aIsNonCompleted && !bIsNonCompleted) return -1
                  if (!aIsNonCompleted && bIsNonCompleted) return 1

                  // Both non-completed: sort by state priority
                  if (aIsNonCompleted && bIsNonCompleted) {
                    return nonCompletedStates.indexOf(a.state) - nonCompletedStates.indexOf(b.state)
                  }

                  // Both completed: sort by return (equity) descending
                  return b.summary.equity_last - a.summary.equity_last
                }).map((run) => (
                  <button
                    key={run.run_id}
                    onClick={() => setSelectedRunId(run.run_id)}
                    className="w-full p-3 rounded-lg text-left transition-all"
                    style={{
                      background: run.run_id === selectedRunId ? 'rgba(240,185,11,0.1)' : '#1E2329',
                      border: `1px solid ${run.run_id === selectedRunId ? '#F0B90B' : '#2B3139'}`,
                    }}
                  >
                    <div className="flex items-center justify-between">
                      <span className="font-mono text-xs" style={{ color: '#EAECEF' }}>
                        {run.run_id.slice(0, 20)}...
                      </span>
                      <span
                        className="flex items-center gap-1 text-xs"
                        style={{ color: getStateColor(run.state) }}
                      >
                        {getStateIcon(run.state)}
                        {tr(`states.${run.state}`)}
                      </span>
                    </div>
                    <div className="flex items-center justify-between mt-1">
                      <span className="text-xs" style={{ color: '#848E9C' }}>
                        {run.summary.progress_pct.toFixed(0)}% · ${run.summary.equity_last.toFixed(0)}
                      </span>
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          toggleCompare(run.run_id)
                        }}
                        className="p-1 rounded"
                        style={{
                          background: compareRunIds.includes(run.run_id)
                            ? 'rgba(240,185,11,0.2)'
                            : 'transparent',
                        }}
                        title={language === 'zh' ? '添加到对比' : 'Add to compare'}
                      >
                        <Eye
                          className="w-3 h-3"
                          style={{
                            color: compareRunIds.includes(run.run_id) ? '#F0B90B' : '#5E6673',
                          }}
                        />
                      </button>
                    </div>
                  </button>
                ))
              )}
            </div>
          </div>
        </div>

        {/* Right Panel - Results */}
        <div className="xl:col-span-2 space-y-4">
          {!selectedRunId ? (
            <div
              className="binance-card p-12 text-center"
              style={{ color: '#5E6673' }}
            >
              <Brain className="w-12 h-12 mx-auto mb-4 opacity-30" />
              <p>{tr('emptyStates.selectRun')}</p>
            </div>
          ) : (
            <>
              {/* Detailed Run Card */}
              {selectedRun && (
                <BacktestRunCard
                  run={selectedRun}
                  config={backtestConfig}
                  metrics={metrics}
                  strategyName={strategies?.find(s => s.id === backtestConfig?.strategy_id)?.name}
                  isSelected={true}
                  onPause={() => handleControl('pause')}
                  onResume={() => handleControl('resume')}
                  onForceResume={handleForceResume}
                  onStop={() => handleControl('stop')}
                  onDelete={() => handleDelete(selectedRun.run_id)}
                  onExport={handleExport}
                  language={language}
                />
              )}

              {/* Real-time Positions Display */}
              {(() => {
                const hasAIPositions = status?.positions && status.positions.length > 0
                // Extract baseline positions from latest decision
                const latestBaselineDecision = baselineDecisions && baselineDecisions.length > 0
                  ? baselineDecisions[baselineDecisions.length - 1]
                  : null
                const baselinePositions = latestBaselineDecision?.positions_json
                  ? JSON.parse(latestBaselineDecision.positions_json)
                  : []
                const hasBaselinePositions = baselinePositions.length > 0

                if (!hasAIPositions && !hasBaselinePositions) {
                  return null
                }

                return (
                  <div className="binance-card p-4">
                    <PositionsDisplay
                      positions={status?.positions || []}
                      baselinePositions={baselinePositions}
                      language={language}
                    />
                  </div>
                )
              })()}

              {/* Tabs */}
              <div className="binance-card">
                <div className="flex border-b" style={{ borderColor: '#2B3139' }}>
                  {(['overview', 'chart', 'trades', 'decisions', 'baseline_decisions'] as ViewTab[]).map((tab) => (
                    <button
                      key={tab}
                      onClick={() => setViewTab(tab)}
                      className="px-4 py-3 text-sm font-medium transition-all relative"
                      style={{ color: viewTab === tab ? '#F0B90B' : '#848E9C' }}
                    >
                      {tab === 'overview'
                        ? language === 'zh'
                          ? '概览'
                          : 'Overview'
                        : tab === 'chart'
                          ? language === 'zh'
                            ? '图表'
                            : 'Chart'
                          : tab === 'trades'
                            ? language === 'zh'
                              ? '交易'
                              : 'Trades'
                            : tab === 'decisions'
                              ? language === 'zh'
                                ? 'AI决策'
                                : 'AI Decisions'
                              : tab === 'baseline_decisions'
                                ? language === 'zh'
                                  ? 'Baseline决策'
                                  : 'Baseline Decisions'
                                : ''}
                      {viewTab === tab && (
                        <motion.div
                          layoutId="tab-indicator"
                          className="absolute bottom-0 left-0 right-0 h-0.5"
                          style={{ background: '#F0B90B' }}
                        />
                      )}
                    </button>
                  ))}
                </div>

                <div className="p-4">
                  <AnimatePresence mode="wait">
                    {viewTab === 'overview' && (
                      <motion.div
                        key="overview"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                      >
                        {equity && equity.length > 0 ? (
                          <BacktestChart equity={equity} trades={trades ?? []} baselineEquity={baselineEquity} />
                        ) : (
                          <div className="py-12 text-center" style={{ color: '#5E6673' }}>
                            {tr('charts.equityEmpty')}
                          </div>
                        )}

                        {metrics && (
                          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mt-4">
                            <div className="p-3 rounded-lg" style={{ background: '#1E2329' }}>
                              <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                                {language === 'zh' ? '胜率' : 'Win Rate'}
                              </div>
                              <div className="text-lg font-bold" style={{ color: '#EAECEF' }}>
                                {(metrics.win_rate ?? 0).toFixed(1)}%
                              </div>
                            </div>
                            <div className="p-3 rounded-lg" style={{ background: '#1E2329' }}>
                              <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                                {language === 'zh' ? '盈亏因子' : 'Profit Factor'}
                              </div>
                              <div className="text-lg font-bold" style={{ color: '#EAECEF' }}>
                                {(metrics.profit_factor ?? 0).toFixed(2)}
                              </div>
                            </div>
                            <div className="p-3 rounded-lg" style={{ background: '#1E2329' }}>
                              <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                                {language === 'zh' ? '总交易数' : 'Total Trades'}
                              </div>
                              <div className="text-lg font-bold" style={{ color: '#EAECEF' }}>
                                {metrics.trades ?? 0}
                              </div>
                            </div>
                            <div className="p-3 rounded-lg" style={{ background: '#1E2329' }}>
                              <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                                {language === 'zh' ? '最佳币种' : 'Best Symbol'}
                              </div>
                              <div className="text-lg font-bold" style={{ color: '#0ECB81' }}>
                                {metrics.best_symbol?.replace('USDT', '') || '-'}
                              </div>
                            </div>
                          </div>
                        )}

                        {/* Baseline vs AI Comparison */}
                        {baselineDecisions && baselineDecisions.length > 0 && (() => {
                          const latestBaseline = baselineDecisions[baselineDecisions.length - 1]
                          const aiEquity = status?.equity || 1000
                          const aiPnlPct = ((aiEquity - 1000) / 1000) * 100
                          const baselinePnlPct = latestBaseline.total_pnl_pct || 0

                          return (
                            <div className="mt-4 p-4 rounded-lg" style={{ background: 'rgba(255, 107, 53, 0.05)', border: '1px solid rgba(255, 107, 53, 0.2)' }}>
                              <div className="flex items-center gap-2 mb-3">
                                <TrendingUp className="w-4 h-4" style={{ color: '#FF6B35' }} />
                                <span className="text-sm font-medium" style={{ color: '#EAECEF' }}>
                                  {language === 'zh' ? 'AI vs Baseline 对比' : 'AI vs Baseline Comparison'}
                                </span>
                              </div>

                              <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                                <div className="p-3 rounded-lg" style={{ background: '#1E2329' }}>
                                  <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                                    {language === 'zh' ? 'AI 收益率' : 'AI Return'}
                                  </div>
                                  <div className="flex items-center gap-2">
                                    <span className="px-1.5 py-0.5 rounded text-[10px]" style={{ background: '#3B82F620', color: '#3B82F6' }}>AI</span>
                                    <span className="text-lg font-bold" style={{ color: aiPnlPct >= 0 ? '#0ECB81' : '#F6465D' }}>
                                      {aiPnlPct >= 0 ? '+' : ''}{aiPnlPct.toFixed(2)}%
                                    </span>
                                  </div>
                                </div>

                                <div className="p-3 rounded-lg" style={{ background: '#1E2329' }}>
                                  <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                                    {language === 'zh' ? 'Baseline 收益率' : 'Baseline Return'}
                                  </div>
                                  <div className="flex items-center gap-2">
                                    <span className="px-1.5 py-0.5 rounded text-[10px]" style={{ background: '#FF6B3520', color: '#FF6B35' }}>Base</span>
                                    <span className="text-lg font-bold" style={{ color: baselinePnlPct >= 0 ? '#0ECB81' : '#F6465D' }}>
                                      {baselinePnlPct >= 0 ? '+' : ''}{baselinePnlPct.toFixed(2)}%
                                    </span>
                                  </div>
                                </div>

                                <div className="p-3 rounded-lg" style={{ background: '#1E2329' }}>
                                  <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                                    {language === 'zh' ? '超额收益' : 'Alpha'}
                                  </div>
                                  <div className="text-lg font-bold" style={{ color: (aiPnlPct - baselinePnlPct) >= 0 ? '#0ECB81' : '#F6465D' }}>
                                    {(aiPnlPct - baselinePnlPct) >= 0 ? '+' : ''}{(aiPnlPct - baselinePnlPct).toFixed(2)}%
                                  </div>
                                </div>

                                <div className="p-3 rounded-lg" style={{ background: '#1E2329' }}>
                                  <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                                    {language === 'zh' ? 'Baseline 最大回撤' : 'Baseline Max DD'}
                                  </div>
                                  <div className="text-lg font-bold" style={{ color: '#F6465D' }}>
                                    -{(latestBaseline.max_drawdown || 0).toFixed(2)}%
                                  </div>
                                </div>
                              </div>
                            </div>
                          )
                        })()}
                      </motion.div>
                    )}

                    {viewTab === 'chart' && (
                      <motion.div
                        key="chart"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        className="space-y-6"
                      >
                        {/* Equity Chart */}
                        <div>
                          <h4 className="text-sm font-medium mb-3" style={{ color: '#EAECEF' }}>
                            {language === 'zh' ? '资金曲线' : 'Equity Curve'}
                          </h4>
                          {equity && equity.length > 0 ? (
                            <BacktestChart equity={equity} trades={trades ?? []} baselineEquity={baselineEquity} />
                          ) : (
                            <div className="py-12 text-center" style={{ color: '#5E6673' }}>
                              {tr('charts.equityEmpty')}
                            </div>
                          )}
                        </div>

                        {/* Candlestick Chart with Trade Markers */}
                        {selectedRunId && trades && trades.length > 0 && (
                          <div>
                            <h4 className="text-sm font-medium mb-3" style={{ color: '#EAECEF' }}>
                              {language === 'zh' ? 'K线图 & 交易标记' : 'Candlestick & Trade Markers'}
                            </h4>
                            <CandlestickChartComponent
                              runId={selectedRunId}
                              trades={trades}
                              language={language}
                            />
                          </div>
                        )}
                      </motion.div>
                    )}

                    {viewTab === 'trades' && (
                      <motion.div
                        key="trades"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                      >
                        <TradeTimeline trades={trades ?? []} />
                      </motion.div>
                    )}

                    {viewTab === 'decisions' && (
                      <motion.div
                        key="decisions"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        className="space-y-3 max-h-[500px] overflow-y-auto"
                      >
                        {decisions && decisions.length > 0 ? (
                          decisions.map((d) => (
                            <DecisionCard
                              key={`${d.cycle_number}-${d.timestamp}`}
                              decision={d}
                              language={language}
                            />
                          ))
                        ) : (
                          <div className="py-12 text-center" style={{ color: '#5E6673' }}>
                            {tr('decisionTrail.emptyHint')}
                          </div>
                        )}
                      </motion.div>
                    )}

                    {viewTab === 'baseline_decisions' && (
                      <motion.div
                        key="baseline_decisions"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        className="space-y-3 max-h-[500px] overflow-y-auto"
                      >
                        {baselineDecisions && baselineDecisions.length > 0 ? (
                          baselineDecisions.map((d, idx) => (
                            <div
                              key={`baseline-${d.cycle}-${idx}`}
                              className="p-4 rounded-lg"
                              style={{ background: '#1E2329', border: '1px solid #2B3139' }}
                            >
                              <div className="flex items-start justify-between mb-3">
                                <div className="flex items-center gap-2">
                                  <Activity className="w-4 h-4" style={{ color: '#FF6B35' }} />
                                  <span className="text-sm font-medium" style={{ color: '#EAECEF' }}>
                                    {language === 'zh' ? '周期' : 'Cycle'} #{d.cycle}
                                  </span>
                                </div>
                                <span className="text-xs" style={{ color: '#848E9C' }}>
                                  {new Date(d.timestamp * 1000).toLocaleString()}
                                </span>
                              </div>

                              <div className="grid grid-cols-4 gap-3 mb-3">
                                <div>
                                  <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                                    {language === 'zh' ? '权益' : 'Equity'}
                                  </div>
                                  <div className="text-sm font-medium" style={{ color: '#EAECEF' }}>
                                    ${d.equity?.toFixed(2) || '0.00'}
                                  </div>
                                </div>
                                <div>
                                  <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                                    {language === 'zh' ? '可用资金' : 'Available'}
                                  </div>
                                  <div className="text-sm font-medium" style={{ color: '#EAECEF' }}>
                                    ${d.available?.toFixed(2) || '0.00'}
                                  </div>
                                </div>
                                <div>
                                  <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                                    {language === 'zh' ? '总盈亏' : 'Total PnL'}
                                  </div>
                                  <div className="text-sm font-medium" style={{ color: (d.total_pnl_pct || 0) >= 0 ? '#0ECB81' : '#F6465D' }}>
                                    {(d.total_pnl_pct || 0) >= 0 ? '+' : ''}{d.total_pnl_pct?.toFixed(2) || '0.00'}%
                                  </div>
                                </div>
                                <div>
                                  <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                                    {language === 'zh' ? '最大回撤' : 'Max DD'}
                                  </div>
                                  <div className="text-sm font-medium" style={{ color: '#F6465D' }}>
                                    {d.max_drawdown?.toFixed(2) || '0.00'}%
                                  </div>
                                </div>
                              </div>

                              {d.reasoning && (
                                <div className="mb-3">
                                  <div className="text-xs mb-1" style={{ color: '#848E9C' }}>
                                    {language === 'zh' ? '决策理由' : 'Reasoning'}
                                  </div>
                                  <div className="text-sm" style={{ color: '#EAECEF' }}>
                                    {d.reasoning}
                                  </div>
                                </div>
                              )}

                              {d.signal_count > 0 && (
                                <div className="flex items-center gap-2 mb-2">
                                  <Zap className="w-3 h-3" style={{ color: '#F0B90B' }} />
                                  <span className="text-xs" style={{ color: '#848E9C' }}>
                                    {language === 'zh' ? '信号数量' : 'Signals'}: {d.signal_count}
                                  </span>
                                </div>
                              )}

                              {d.position_count > 0 && (
                                <div className="mb-3 p-3 rounded" style={{ background: '#0B0E11', border: '1px solid #2B3139' }}>
                                  <div className="flex items-center justify-between mb-2">
                                    <span className="text-xs font-medium" style={{ color: '#EAECEF' }}>
                                      {language === 'zh' ? '当前持仓' : 'Current Positions'} ({d.position_count})
                                    </span>
                                    <span className="text-xs" style={{ color: '#848E9C' }}>
                                      {language === 'zh' ? '总价值' : 'Total'}: ${d.position_value?.toFixed(2) || '0.00'}
                                    </span>
                                  </div>
                                  {d.positions_json && d.positions_json !== '[]' && (
                                    <div className="space-y-2">
                                      {JSON.parse(d.positions_json).map((pos: any, pidx: number) => (
                                        <div key={pidx} className="text-xs p-2 rounded" style={{ background: '#1E2329' }}>
                                          <div className="flex items-center justify-between">
                                            <span style={{ color: '#EAECEF' }}>{pos.symbol}</span>
                                            <span style={{ color: pos.side === 'long' ? '#0ECB81' : '#F6465D' }}>
                                              {pos.side === 'long' ? '多' : '空'}
                                            </span>
                                          </div>
                                        </div>
                                      ))}
                                    </div>
                                  )}
                                </div>
                              )}

                              {d.decision_json && d.decision_json !== '[]' && (
                                <div className="mt-3 pt-3" style={{ borderTop: '1px solid #2B3139' }}>
                                  <div className="text-xs mb-2" style={{ color: '#848E9C' }}>
                                    {language === 'zh' ? '决策详情' : 'Decision Details'}
                                  </div>
                                  <pre className="text-xs p-2 rounded overflow-x-auto" style={{ background: '#0B0E11', color: '#848E9C' }}>
                                    {JSON.stringify(JSON.parse(d.decision_json), null, 2)}
                                  </pre>
                                </div>
                              )}
                            </div>
                          ))
                        ) : (
                          <div className="py-12 text-center" style={{ color: '#5E6673' }}>
                            {language === 'zh' ? '暂无 Baseline 决策记录' : 'No Baseline decisions yet'}
                          </div>
                        )}
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
