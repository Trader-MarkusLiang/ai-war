import { Play, Pause, Square, TrendingUp, TrendingDown } from 'lucide-react'
import type { Evolution } from '../../types'

interface EvolutionCardProps {
  evolution: Evolution
  onStart: (id: string) => void
  onPause: (id: string) => void
  onStop: (id: string) => void
  onSelect: (evolution: Evolution) => void
}

export default function EvolutionCard({
  evolution,
  onStart,
  onPause,
  onStop,
  onSelect,
}: EvolutionCardProps) {
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'bg-green-100 text-green-800'
      case 'paused':
        return 'bg-yellow-100 text-yellow-800'
      case 'completed':
        return 'bg-blue-100 text-blue-800'
      case 'stopped':
        return 'bg-gray-100 text-gray-800'
      default:
        return 'bg-gray-100 text-gray-800'
    }
  }

  const getStatusText = (status: string) => {
    switch (status) {
      case 'running':
        return 'Running'
      case 'paused':
        return 'Paused'
      case 'completed':
        return 'Completed'
      case 'stopped':
        return 'Stopped'
      case 'created':
        return 'Created'
      default:
        return status
    }
  }

  const progress = (evolution.current_iteration / evolution.max_iterations) * 100

  return (
    <div
      className="bg-white rounded-lg border border-gray-200 p-4 hover:shadow-md transition-shadow cursor-pointer"
      onClick={() => onSelect(evolution)}
    >
      {/* Header */}
      <div className="flex items-start justify-between mb-3">
        <div className="flex-1">
          <h3 className="font-semibold text-gray-900">{evolution.name}</h3>
          <p className="text-xs text-gray-500 mt-1">
            Base Strategy: {evolution.base_strategy_id.slice(0, 8)}...
          </p>
          {evolution.ai_model_name && (
            <p className="text-xs text-blue-600 mt-0.5">
              AI: {evolution.ai_model_name}
            </p>
          )}
        </div>
        <span className={`px-2 py-1 rounded text-xs font-medium ${getStatusColor(evolution.status)}`}>
          {getStatusText(evolution.status)}
        </span>
      </div>

      {/* Progress */}
      <div className="mb-3">
        <div className="flex justify-between text-xs text-gray-600 mb-1">
          <span>
            Iteration {evolution.current_iteration} / {evolution.max_iterations}
          </span>
          <span>{progress.toFixed(0)}%</span>
        </div>
        <div className="w-full bg-gray-200 rounded-full h-2">
          <div
            className="bg-blue-600 h-2 rounded-full transition-all"
            style={{ width: `${progress}%` }}
          ></div>
        </div>
        {evolution.status === 'running' && evolution.backtest_progress > 0 && (
          <div className="mt-1">
            <div className="flex justify-between text-xs text-gray-500">
              <span className="truncate max-w-[120px]" title={evolution.current_backtest_id}>
                {evolution.current_backtest_id || 'Backtest'}
              </span>
              <span>{evolution.backtest_progress.toFixed(1)}%</span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-1 mt-0.5">
              <div
                className="bg-green-500 h-1 rounded-full transition-all"
                style={{ width: `${evolution.backtest_progress}%` }}
              ></div>
            </div>
            {/* Real-time metrics */}
            {evolution.current_equity !== undefined && evolution.current_equity > 0 && (
              <div className="grid grid-cols-3 gap-1 mt-2 text-[10px]">
                <div className="bg-gray-100 rounded px-1.5 py-1 text-center">
                  <p className="text-gray-500">资金</p>
                  <p className="font-medium text-gray-800">{evolution.current_equity.toFixed(0)}</p>
                </div>
                <div className="bg-gray-100 rounded px-1.5 py-1 text-center">
                  <p className="text-gray-500">收益</p>
                  <p className={`font-medium ${(evolution.current_return_pct ?? 0) >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                    {(evolution.current_return_pct ?? 0) >= 0 ? '+' : ''}{(evolution.current_return_pct ?? 0).toFixed(2)}%
                  </p>
                </div>
                <div className="bg-gray-100 rounded px-1.5 py-1 text-center">
                  <p className="text-gray-500">回撤</p>
                  <p className="font-medium text-red-600">{(evolution.current_drawdown ?? 0).toFixed(2)}%</p>
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Iteration Timeline */}
      {evolution.iterations && evolution.iterations.length > 0 && (
        <div className="mb-3">
          <div className="flex items-center justify-between mb-2">
            <p className="text-xs text-gray-500">Iterations</p>
            {/* Current Status Display */}
            {evolution.status === 'running' && (() => {
              const currentIter = evolution.iterations.find(
                (iter) => ['backtest', 'evaluating', 'optimizing'].includes(iter.status)
              )
              if (currentIter) {
                const statusConfig: Record<string, { text: string; color: string }> = {
                  backtest: { text: 'Backtesting...', color: 'text-blue-600' },
                  evaluating: { text: 'AI Evaluating...', color: 'text-purple-600' },
                  optimizing: { text: 'AI Optimizing...', color: 'text-orange-600' },
                }
                const config = statusConfig[currentIter.status]
                return config ? (
                  <span className={`text-xs font-medium ${config.color} animate-pulse`}>
                    v{currentIter.version}: {config.text}
                  </span>
                ) : null
              }
              return null
            })()}
          </div>
          <div className="flex gap-1 flex-wrap">
            {evolution.iterations.map((iter) => (
              <div
                key={iter.version}
                className="relative group"
                title={`v${iter.version}: ${iter.status}${iter.metrics ? ` (${iter.metrics.total_return >= 0 ? '+' : ''}${iter.metrics.total_return.toFixed(1)}%)` : ''}`}
              >
                <div
                  className={`w-6 h-6 rounded flex items-center justify-center text-[10px] font-medium ${
                    iter.status === 'completed'
                      ? iter.metrics && iter.metrics.total_return >= 0
                        ? 'bg-green-100 text-green-700'
                        : 'bg-red-100 text-red-700'
                      : iter.status === 'failed'
                      ? 'bg-red-100 text-red-700'
                      : ['backtest', 'evaluating', 'optimizing'].includes(iter.status)
                      ? 'bg-blue-100 text-blue-700'
                      : 'bg-gray-100 text-gray-500'
                  }`}
                >
                  {iter.version}
                </div>
                {iter.version === evolution.best_version && iter.status === 'completed' && (
                  <div className="absolute -top-1 -right-1 w-2 h-2 bg-yellow-400 rounded-full border border-white" />
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Metrics */}
      <div className="grid grid-cols-2 gap-2 mb-3">
        <div className="bg-gray-50 rounded p-2">
          <p className="text-xs text-gray-500">Best Version</p>
          <p className="text-sm font-semibold text-gray-900">v{evolution.best_version}</p>
        </div>
        <div className="bg-gray-50 rounded p-2">
          <p className="text-xs text-gray-500">Best Return</p>
          <div className="flex items-center gap-1">
            {evolution.best_return >= 0 ? (
              <TrendingUp className="w-3 h-3 text-green-600" />
            ) : (
              <TrendingDown className="w-3 h-3 text-red-600" />
            )}
            <p
              className={`text-sm font-semibold ${
                evolution.best_return >= 0 ? 'text-green-600' : 'text-red-600'
              }`}
            >
              {evolution.best_return.toFixed(2)}%
            </p>
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="flex gap-2" onClick={(e) => e.stopPropagation()}>
        {evolution.status === 'created' || evolution.status === 'paused' || evolution.status === 'stopped' ? (
          <button
            onClick={() => onStart(evolution.id)}
            className="flex-1 px-3 py-1.5 bg-green-600 text-white rounded text-sm hover:bg-green-700 flex items-center justify-center gap-1"
          >
            <Play className="w-3 h-3" />
            {evolution.status === 'stopped' ? 'Resume' : 'Start'}
          </button>
        ) : null}
        {evolution.status === 'running' ? (
          <button
            onClick={() => onPause(evolution.id)}
            className="flex-1 px-3 py-1.5 bg-yellow-600 text-white rounded text-sm hover:bg-yellow-700 flex items-center justify-center gap-1"
          >
            <Pause className="w-3 h-3" />
            Pause
          </button>
        ) : null}
        {evolution.status === 'running' || evolution.status === 'paused' ? (
          <button
            onClick={() => onStop(evolution.id)}
            className="flex-1 px-3 py-1.5 bg-red-600 text-white rounded text-sm hover:bg-red-700 flex items-center justify-center gap-1"
          >
            <Square className="w-3 h-3" />
            Stop
          </button>
        ) : null}
      </div>
    </div>
  )
}
