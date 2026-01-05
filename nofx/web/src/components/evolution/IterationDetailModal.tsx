import { useState, useEffect } from 'react'
import { X, Loader2 } from 'lucide-react'
import { api } from '../../lib/api'
import type { EvolutionIterationDetail } from '../../types'

interface IterationDetailModalProps {
  evolutionId: string
  version: number
  isOpen: boolean
  onClose: () => void
}

export default function IterationDetailModal({
  evolutionId,
  version,
  isOpen,
  onClose,
}: IterationDetailModalProps) {
  const [iteration, setIteration] = useState<EvolutionIterationDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<'metrics' | 'evaluation' | 'prompt'>('metrics')

  useEffect(() => {
    if (isOpen && evolutionId && version) {
      loadIteration()
    }
  }, [isOpen, evolutionId, version])

  const loadIteration = async () => {
    try {
      setLoading(true)
      const data = await api.getEvolutionIteration(evolutionId, version)
      setIteration(data)
    } catch (error) {
      console.error('Failed to load iteration:', error)
    } finally {
      setLoading(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b">
          <div>
            <h2 className="text-xl font-bold text-gray-900">Iteration v{version}</h2>
            <p className="text-sm text-gray-500 mt-1">Detailed analysis and metrics</p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Tabs */}
        <div className="flex border-b">
          <button
            onClick={() => setActiveTab('metrics')}
            className={`px-6 py-3 text-sm font-medium border-b-2 transition-colors ${
              activeTab === 'metrics'
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            Metrics
          </button>
          <button
            onClick={() => setActiveTab('evaluation')}
            className={`px-6 py-3 text-sm font-medium border-b-2 transition-colors ${
              activeTab === 'evaluation'
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            AI Evaluation
          </button>
          <button
            onClick={() => setActiveTab('prompt')}
            className={`px-6 py-3 text-sm font-medium border-b-2 transition-colors ${
              activeTab === 'prompt'
                ? 'border-blue-600 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700'
            }`}
          >
            Prompt Changes
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
            </div>
          ) : iteration ? (
            <>
              {/* Metrics Tab */}
              {activeTab === 'metrics' && (
                <div className="space-y-4">
                  <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                    {iteration.metrics && (
                      <>
                        <MetricCard
                          label="Total Return"
                          value={`${iteration.metrics.total_return.toFixed(2)}%`}
                          positive={iteration.metrics.total_return >= 0}
                        />
                        <MetricCard
                          label="Max Drawdown"
                          value={`${iteration.metrics.max_drawdown.toFixed(2)}%`}
                          negative
                        />
                        <MetricCard
                          label="Win Rate"
                          value={`${iteration.metrics.win_rate.toFixed(1)}%`}
                        />
                        <MetricCard
                          label="Sharpe Ratio"
                          value={iteration.metrics.sharpe_ratio.toFixed(2)}
                        />
                        <MetricCard
                          label="Total Trades"
                          value={iteration.metrics.trades.toString()}
                        />
                      </>
                    )}
                  </div>
                  {iteration.changes_summary && (
                    <div className="mt-6">
                      <h3 className="text-sm font-semibold text-gray-900 mb-2">Changes Summary</h3>
                      <p className="text-sm text-gray-700 whitespace-pre-wrap">{iteration.changes_summary}</p>
                    </div>
                  )}
                </div>
              )}

              {/* Evaluation Tab */}
              {activeTab === 'evaluation' && (
                <div className="space-y-4">
                  {iteration.evaluation ? (
                    <>
                      <div>
                        <h3 className="text-sm font-semibold text-green-700 mb-2">✓ Strengths</h3>
                        <ul className="list-disc list-inside space-y-1">
                          {iteration.evaluation.strengths.map((item, i) => (
                            <li key={i} className="text-sm text-gray-700">{item}</li>
                          ))}
                        </ul>
                      </div>
                      <div>
                        <h3 className="text-sm font-semibold text-red-700 mb-2">✗ Weaknesses</h3>
                        <ul className="list-disc list-inside space-y-1">
                          {iteration.evaluation.weaknesses.map((item, i) => (
                            <li key={i} className="text-sm text-gray-700">{item}</li>
                          ))}
                        </ul>
                      </div>
                      <div>
                        <h3 className="text-sm font-semibold text-blue-700 mb-2">💡 Suggestions</h3>
                        <ul className="list-disc list-inside space-y-1">
                          {iteration.evaluation.suggestions.map((item, i) => (
                            <li key={i} className="text-sm text-gray-700">{item}</li>
                          ))}
                        </ul>
                      </div>
                    </>
                  ) : (
                    <p className="text-sm text-gray-500">No evaluation available</p>
                  )}
                </div>
              )}

              {/* Prompt Tab */}
              {activeTab === 'prompt' && (
                <div className="space-y-4">
                  {iteration.prompt_before && iteration.prompt_after ? (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <h3 className="text-sm font-semibold text-gray-900 mb-2">Before</h3>
                        <pre className="text-xs bg-gray-50 p-4 rounded border overflow-auto max-h-96">
                          {iteration.prompt_before}
                        </pre>
                      </div>
                      <div>
                        <h3 className="text-sm font-semibold text-gray-900 mb-2">After</h3>
                        <pre className="text-xs bg-gray-50 p-4 rounded border overflow-auto max-h-96">
                          {iteration.prompt_after}
                        </pre>
                      </div>
                    </div>
                  ) : (
                    <p className="text-sm text-gray-500">No prompt changes available</p>
                  )}
                </div>
              )}
            </>
          ) : (
            <div className="text-center py-12 text-gray-500">
              Failed to load iteration details
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function MetricCard({
  label,
  value,
  positive,
  negative,
}: {
  label: string
  value: string
  positive?: boolean
  negative?: boolean
}) {
  return (
    <div className="bg-gray-50 rounded-lg p-4">
      <p className="text-xs text-gray-500 mb-1">{label}</p>
      <p
        className={`text-lg font-bold ${
          positive
            ? 'text-green-600'
            : negative
            ? 'text-red-600'
            : 'text-gray-900'
        }`}
      >
        {value}
      </p>
    </div>
  )
}
