import { useMemo } from 'react'
import { motion } from 'framer-motion'
import {
  Download,
  Trash2,
  Play,
  Pause,
  Square,
  Clock,
  TrendingUp,
  TrendingDown,
  AlertTriangle,
  Zap,
  Database,
  BarChart3,
  Cpu,
  Layers,
  Calendar,
  RefreshCw,
} from 'lucide-react'
import type {
  BacktestRunMetadata,
  BacktestStartConfig,
  BacktestMetrics,
} from '../types'

// Props interface
interface BacktestRunCardProps {
  run: BacktestRunMetadata
  config?: BacktestStartConfig
  metrics?: BacktestMetrics
  strategyName?: string
  isSelected?: boolean
  onSelect?: () => void
  onPause?: () => void
  onResume?: () => void
  onForceResume?: () => void
  onStop?: () => void
  onDelete?: () => void
  onExport?: () => void
  language?: string
}

// Progress Ring Component
function ProgressRing({ progress, size = 64 }: { progress: number; size?: number }) {
  const strokeWidth = 5
  const radius = (size - strokeWidth) / 2
  const circumference = radius * 2 * Math.PI
  const offset = circumference - (progress / 100) * circumference

  // Color based on progress
  const getColor = () => {
    if (progress >= 100) return '#0ECB81'
    if (progress >= 50) return '#F0B90B'
    return '#3B82F6'
  }

  return (
    <div className="relative" style={{ width: size, height: size }}>
      <svg className="transform -rotate-90" width={size} height={size}>
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          stroke="#2B3139"
          strokeWidth={strokeWidth}
          fill="none"
        />
        <motion.circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          stroke={getColor()}
          strokeWidth={strokeWidth}
          fill="none"
          strokeLinecap="round"
          strokeDasharray={circumference}
          initial={{ strokeDashoffset: circumference }}
          animate={{ strokeDashoffset: offset }}
          transition={{ duration: 0.5 }}
        />
      </svg>
      <div className="absolute inset-0 flex items-center justify-center flex-col">
        <span className="text-sm font-bold" style={{ color: getColor() }}>
          {progress.toFixed(0)}%
        </span>
        <span className="text-[10px]" style={{ color: '#848E9C' }}>
          Complete
        </span>
      </div>
    </div>
  )
}

// State badge component
function StateBadge({ state }: { state: string }) {
  const getStateStyle = () => {
    switch (state) {
      case 'running':
        return { bg: 'rgba(240, 185, 11, 0.15)', color: '#F0B90B', text: 'Running' }
      case 'completed':
        return { bg: 'rgba(14, 203, 129, 0.15)', color: '#0ECB81', text: 'Completed' }
      case 'failed':
      case 'liquidated':
        return { bg: 'rgba(246, 70, 93, 0.15)', color: '#F6465D', text: state === 'liquidated' ? 'Liquidated' : 'Failed' }
      case 'paused':
        return { bg: 'rgba(132, 142, 156, 0.15)', color: '#848E9C', text: 'Paused' }
      default:
        return { bg: 'rgba(132, 142, 156, 0.15)', color: '#848E9C', text: state }
    }
  }

  const style = getStateStyle()

  return (
    <span
      className="px-2 py-0.5 rounded text-xs font-medium inline-flex items-center gap-1"
      style={{ background: style.bg, color: style.color }}
    >
      {state === 'running' && <span className="w-1.5 h-1.5 rounded-full bg-current animate-pulse" />}
      {style.text}
    </span>
  )
}

// Format duration helper
function formatDuration(startTs: number, endTs: number): string {
  const diffMs = (endTs - startTs) * 1000
  const days = Math.floor(diffMs / (1000 * 60 * 60 * 24))
  const hours = Math.floor((diffMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))

  if (days > 0) {
    return `${days}d ${hours}h`
  }
  const minutes = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60))
  if (hours > 0) {
    return `${hours}h ${minutes}m`
  }
  return `${minutes}m`
}

// Format date helper
function formatDate(ts: number): string {
  return new Date(ts * 1000).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function BacktestRunCard({
  run,
  config,
  metrics,
  strategyName,
  isSelected,
  onSelect,
  onPause,
  onResume,
  onForceResume,
  onStop,
  onDelete,
  onExport,
  language = 'en',
}: BacktestRunCardProps) {
  const zh = language === 'zh'

  // Derived values
  const progress = run.summary.progress_pct
  const isRunning = run.state === 'running'
  const isPaused = run.state === 'paused'

  // Calculate runtime if config available
  const runtime = useMemo(() => {
    if (!config) return null
    return formatDuration(config.start_ts, config.end_ts)
  }, [config])

  // Format symbols display
  const symbolsDisplay = useMemo(() => {
    if (!config?.symbols) return `${run.summary.symbol_count} symbols`
    if (config.symbols.length <= 3) {
      return config.symbols.map(s => s.replace('USDT', '')).join(', ')
    }
    return `${config.symbols.length} symbols`
  }, [config, run.summary.symbol_count])

  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      className="rounded-xl overflow-hidden cursor-pointer transition-all"
      style={{
        background: '#181A20',
        border: isSelected ? '1px solid #F0B90B' : '1px solid #2B3139',
        boxShadow: isSelected ? '0 0 20px rgba(240, 185, 11, 0.15)' : 'none',
      }}
      onClick={onSelect}
    >
      {/* Header Section */}
      <div className="p-4 pb-3">
        <div className="flex items-start gap-3">
          {/* Progress Ring */}
          <ProgressRing progress={progress} size={64} />

          {/* Title & Status */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <h3
                className="font-mono font-bold text-sm truncate"
                style={{ color: '#EAECEF' }}
                title={run.run_id}
              >
                {run.run_id.length > 28 ? run.run_id.slice(0, 28) + '...' : run.run_id}
              </h3>
              <StateBadge state={run.state} />
            </div>

            {/* Quick Info */}
            <div className="flex items-center gap-3 mt-1.5 text-xs flex-wrap" style={{ color: '#848E9C' }}>
              <span className="flex items-center gap-1">
                <Clock className="w-3 h-3" />
                {run.summary.decision_tf || '5m'}
              </span>
              <span>·</span>
              <span>{symbolsDisplay}</span>
              {runtime && (
                <>
                  <span>·</span>
                  <span className="flex items-center gap-1">
                    <Calendar className="w-3 h-3" />
                    {runtime}
                  </span>
                </>
              )}
            </div>

            {/* Date Range */}
            {config && (
              <div className="text-[10px] mt-1" style={{ color: '#5E6673' }}>
                <Calendar className="w-3 h-3 inline mr-1" />
                {formatDate(config.start_ts)} ~ {formatDate(config.end_ts)}
              </div>
            )}
          </div>

          {/* Action Buttons */}
          <div className="flex items-center gap-1">
            {isRunning && onPause && (
              <button
                onClick={(e) => { e.stopPropagation(); onPause(); }}
                className="p-1.5 rounded-lg transition-colors hover:bg-[#2B3139]"
                title={zh ? '暂停' : 'Pause'}
              >
                <Pause className="w-4 h-4" style={{ color: '#F0B90B' }} />
              </button>
            )}
            {isPaused && onResume && (
              <button
                onClick={(e) => { e.stopPropagation(); onResume(); }}
                className="p-1.5 rounded-lg transition-colors hover:bg-[#2B3139]"
                title={zh ? '继续' : 'Resume'}
              >
                <Play className="w-4 h-4" style={{ color: '#0ECB81' }} />
              </button>
            )}
            {(isRunning || isPaused) && onStop && (
              <button
                onClick={(e) => { e.stopPropagation(); onStop(); }}
                className="p-1.5 rounded-lg transition-colors hover:bg-[#2B3139]"
                title={zh ? '停止' : 'Stop'}
              >
                <Square className="w-4 h-4" style={{ color: '#F6465D' }} />
              </button>
            )}
            {!isRunning && !isPaused && progress < 100 && onForceResume && (
              <button
                onClick={(e) => { e.stopPropagation(); onForceResume(); }}
                className="p-1.5 rounded-lg transition-colors hover:bg-[#2B3139]"
                title={zh ? '强制恢复' : 'Force Resume'}
              >
                <RefreshCw className="w-4 h-4" style={{ color: '#F0B90B' }} />
              </button>
            )}
            {onExport && (
              <button
                onClick={(e) => { e.stopPropagation(); onExport(); }}
                className="p-1.5 rounded-lg transition-colors hover:bg-[#2B3139]"
                title={zh ? '导出' : 'Export'}
              >
                <Download className="w-4 h-4" style={{ color: '#EAECEF' }} />
              </button>
            )}
            {onDelete && (
              <button
                onClick={(e) => { e.stopPropagation(); onDelete(); }}
                className="p-1.5 rounded-lg transition-colors hover:bg-[#2B3139]"
                title={zh ? '删除' : 'Delete'}
              >
                <Trash2 className="w-4 h-4" style={{ color: '#F6465D' }} />
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Parameters Grid */}
      {config && (
        <div
          className="px-4 py-3 grid grid-cols-4 gap-x-4 gap-y-2"
          style={{ background: 'rgba(30, 35, 41, 0.5)', borderTop: '1px solid #2B3139' }}
        >
          <ParamItem
            icon={<Layers className="w-3.5 h-3.5" />}
            label={zh ? '回测币种' : 'Symbols'}
            value={config.symbols?.length > 0 ? config.symbols.map(s => s.replace('USDT', '')).join(', ') : '-'}
          />
          <ParamItem
            icon={<Clock className="w-3.5 h-3.5" />}
            label={zh ? '回测周期' : 'Duration'}
            value={runtime || '-'}
          />
          <ParamItem
            icon={<Zap className="w-3.5 h-3.5" />}
            label={zh ? '决策间隔' : 'Decision'}
            value={`${run.summary.decision_tf || config.decision_timeframe} × ${config.decision_cadence_nbars || 1}`}
          />
          <ParamItem
            icon={<TrendingUp className="w-3.5 h-3.5" />}
            label={zh ? '杠杆' : 'Leverage'}
            value={`${config.leverage?.btc_eth_leverage || 5}x / ${config.leverage?.altcoin_leverage || 5}x`}
          />
          <ParamItem
            icon={<Database className="w-3.5 h-3.5" />}
            label={zh ? '初始资金' : 'Initial'}
            value={`${config.initial_balance?.toLocaleString() || 1000} USDT`}
          />
          <ParamItem
            icon={<Cpu className="w-3.5 h-3.5" />}
            label={zh ? '策略配置' : 'Strategy'}
            value={strategyName || run.label || '-'}
          />
          <ParamItem
            icon={<BarChart3 className="w-3.5 h-3.5" />}
            label={zh ? 'AI 缓存' : 'AI Cache'}
            value={config.cache_ai ? (zh ? '已启用' : 'Enabled') : (zh ? '已禁用' : 'Disabled')}
            valueColor={config.cache_ai ? '#0ECB81' : '#848E9C'}
          />
          <ParamItem
            icon={<AlertTriangle className="w-3.5 h-3.5" />}
            label={zh ? '成交模式' : 'Fill'}
            value={config.fill_policy === 'next_open' ? 'Next Open' : config.fill_policy || 'Next Open'}
          />
        </div>
      )}

      {/* Metrics Section */}
      <div
        className="px-4 py-3 grid grid-cols-4 gap-2"
        style={{ background: '#0B0E11', borderTop: '1px solid #2B3139' }}
      >
        <MetricItem
          label={zh ? '当前净值' : 'Equity'}
          value={`${run.summary.equity_last?.toFixed(2) || '0.00'}`}
          suffix="USDT"
        />
        <MetricItem
          label={zh ? '总收益率' : 'Return'}
          value={`${(metrics?.total_return_pct ?? 0).toFixed(2)}%`}
          trend={(metrics?.total_return_pct ?? 0) >= 0 ? 'up' : 'down'}
        />
        <MetricItem
          label={zh ? '最大回撤' : 'Max DD'}
          value={`${(run.summary.max_drawdown_pct ?? metrics?.max_drawdown_pct ?? 0).toFixed(2)}%`}
          trend="down"
        />
        <MetricItem
          label={zh ? '夏普比率' : 'Sharpe'}
          value={(metrics?.sharpe_ratio ?? 0).toFixed(2)}
        />
      </div>

      {/* Error/Note Display */}
      {(run.last_error || run.summary.liquidation_note) && (
        <div
          className="px-4 py-2 text-xs flex items-center gap-2"
          style={{
            background: 'rgba(246, 70, 93, 0.1)',
            borderTop: '1px solid rgba(246, 70, 93, 0.2)',
            color: '#F6465D',
          }}
        >
          <AlertTriangle className="w-3.5 h-3.5 flex-shrink-0" />
          <span className="truncate">{run.last_error || run.summary.liquidation_note}</span>
        </div>
      )}
    </motion.div>
  )
}

// Parameter Item Component
function ParamItem({
  icon,
  label,
  value,
  valueColor,
}: {
  icon: React.ReactNode
  label: string
  value: string
  valueColor?: string
}) {
  return (
    <div className="flex flex-col">
      <div className="flex items-center gap-1 text-[10px]" style={{ color: '#5E6673' }}>
        {icon}
        <span>{label}</span>
      </div>
      <div
        className="text-xs font-medium mt-0.5 truncate"
        style={{ color: valueColor || '#EAECEF' }}
        title={value}
      >
        {value}
      </div>
    </div>
  )
}

// Metric Item Component
function MetricItem({
  label,
  value,
  suffix,
  trend,
}: {
  label: string
  value: string
  suffix?: string
  trend?: 'up' | 'down'
}) {
  const getColor = () => {
    if (!trend) return '#EAECEF'
    return trend === 'up' ? '#0ECB81' : '#F6465D'
  }

  return (
    <div
      className="p-2 rounded-lg text-center"
      style={{ background: 'rgba(30, 35, 41, 0.6)' }}
    >
      <div className="text-[10px] mb-1" style={{ color: '#5E6673' }}>
        {label}
      </div>
      <div className="flex items-baseline justify-center gap-1">
        <span className="text-sm font-bold" style={{ color: getColor() }}>
          {trend === 'up' && parseFloat(value) > 0 && '+'}
          {value}
        </span>
        {suffix && (
          <span className="text-[10px]" style={{ color: '#848E9C' }}>
            {suffix}
          </span>
        )}
        {trend && (
          <span style={{ color: getColor() }}>
            {trend === 'up' ? (
              <TrendingUp className="w-3 h-3" />
            ) : (
              <TrendingDown className="w-3 h-3" />
            )}
          </span>
        )}
      </div>
    </div>
  )
}

export default BacktestRunCard
