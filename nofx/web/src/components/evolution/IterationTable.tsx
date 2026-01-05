import { useState, useEffect } from 'react'
import { TrendingUp, TrendingDown, ExternalLink } from 'lucide-react'
import { api } from '../../lib/api'
import type { EvolutionIteration } from '../../types'

interface IterationTableProps {
  evolutionId: string
  onSelectIteration: (version: number) => void
}

export default function IterationTable({
  evolutionId,
  onSelectIteration,
}: IterationTableProps) {
  const [iterations, setIterations] = useState<EvolutionIteration[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadIterations()
  }, [evolutionId])

  const loadIterations = async () => {
    try {
      setLoading(true)
      const data = await api.getEvolutionIterations(evolutionId)
      setIterations(data || [])
    } catch (error) {
      console.error('Failed to load iterations:', error)
      setIterations([])
    } finally {
      setLoading(false)
    }
  }

  const getStatusBadge = (status: string) => {
    const colors: Record<string, string> = {
      pending: 'bg-gray-100 text-gray-800',
      backtest: 'bg-blue-100 text-blue-800',
      evaluating: 'bg-purple-100 text-purple-800',
      optimizing: 'bg-yellow-100 text-yellow-800',
      completed: 'bg-green-100 text-green-800',
      failed: 'bg-red-100 text-red-800',
    }
    return colors[status] || colors.pending
  }

  if (loading) {
    return (
      <div className="text-center py-8">
        <div className="inline-block animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Version</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Status</th>
              <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Return</th>
              <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Drawdown</th>
              <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Win Rate</th>
              <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Sharpe</th>
              <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">Trades</th>
              <th className="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {iterations.map((iter) => (
              <tr key={iter.version} className="hover:bg-gray-50">
                <td className="px-4 py-3 text-sm font-medium text-gray-900">
                  v{iter.version}
                </td>
                <td className="px-4 py-3">
                  <span className={`px-2 py-1 rounded text-xs font-medium ${getStatusBadge(iter.status)}`}>
                    {iter.status}
                  </span>
                </td>
                <td className="px-4 py-3 text-sm text-right">
                  {iter.metrics ? (
                    <div className="flex items-center justify-end gap-1">
                      {iter.metrics.total_return >= 0 ? (
                        <TrendingUp className="w-3 h-3 text-green-600" />
                      ) : (
                        <TrendingDown className="w-3 h-3 text-red-600" />
                      )}
                      <span className={iter.metrics.total_return >= 0 ? 'text-green-600' : 'text-red-600'}>
                        {iter.metrics.total_return.toFixed(2)}%
                      </span>
                    </div>
                  ) : (
                    <span className="text-gray-400">-</span>
                  )}
                </td>
                <td className="px-4 py-3 text-sm text-right text-red-600">
                  {iter.metrics ? `${iter.metrics.max_drawdown.toFixed(2)}%` : '-'}
                </td>
                <td className="px-4 py-3 text-sm text-right text-gray-900">
                  {iter.metrics ? `${iter.metrics.win_rate.toFixed(1)}%` : '-'}
                </td>
                <td className="px-4 py-3 text-sm text-right text-gray-900">
                  {iter.metrics ? iter.metrics.sharpe_ratio.toFixed(2) : '-'}
                </td>
                <td className="px-4 py-3 text-sm text-right text-gray-900">
                  {iter.metrics ? iter.metrics.trades : '-'}
                </td>
                <td className="px-4 py-3 text-center">
                  <button
                    onClick={() => onSelectIteration(iter.version)}
                    className="text-blue-600 hover:text-blue-800"
                    title="View details"
                  >
                    <ExternalLink className="w-4 h-4" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {iterations.length === 0 && (
        <div className="text-center py-8 text-gray-500">
          No iterations yet
        </div>
      )}
    </div>
  )
}
