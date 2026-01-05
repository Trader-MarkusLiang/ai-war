import { useState } from 'react'
import { X, TrendingUp, TrendingDown, Play, Pause, Square, Trash2, RotateCcw } from 'lucide-react'
import { api } from '../../lib/api'
import type { Evolution } from '../../types'
import IterationTable from './IterationTable'
import IterationDetailModal from './IterationDetailModal'

interface EvolutionDetailModalProps {
  evolution: Evolution | null
  isOpen: boolean
  onClose: () => void
  onRefresh?: () => void
}

export default function EvolutionDetailModal({
  evolution,
  isOpen,
  onClose,
  onRefresh,
}: EvolutionDetailModalProps) {
  const [selectedVersion, setSelectedVersion] = useState<number | null>(null)
  const [showIterationDetail, setShowIterationDetail] = useState(false)
  const [loading, setLoading] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const handleSelectIteration = (version: number) => {
    setSelectedVersion(version)
    setShowIterationDetail(true)
  }

  const handleAction = async (action: 'start' | 'pause' | 'resume' | 'stop') => {
    if (!evolution) return
    setLoading(true)
    try {
      switch (action) {
        case 'start':
          await api.startEvolution(evolution.id)
          break
        case 'pause':
          await api.pauseEvolution(evolution.id)
          break
        case 'resume':
          await api.resumeEvolution(evolution.id)
          break
        case 'stop':
          await api.stopEvolution(evolution.id)
          break
      }
      onRefresh?.()
    } catch (error) {
      console.error(`Failed to ${action} evolution:`, error)
    } finally {
      setLoading(false)
    }
  }

  const handleDelete = async () => {
    if (!evolution) return
    setLoading(true)
    try {
      await api.deleteEvolution(evolution.id)
      onRefresh?.()
      onClose()
    } catch (error) {
      console.error('Failed to delete evolution:', error)
    } finally {
      setLoading(false)
      setShowDeleteConfirm(false)
    }
  }

  if (!isOpen || !evolution) return null

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

  const progress = (evolution.current_iteration / evolution.max_iterations) * 100

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg shadow-xl max-w-6xl w-full max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b">
          <div className="flex-1">
            <h2 className="text-2xl font-bold text-gray-900">{evolution.name}</h2>
            <p className="text-sm text-gray-500 mt-1">
              Evolution ID: {evolution.id.slice(0, 8)}...
            </p>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 ml-4"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {/* Status Overview */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="bg-gray-50 rounded-lg p-4">
              <p className="text-xs text-gray-500 mb-1">Status</p>
              <span className={`inline-block px-2 py-1 rounded text-xs font-medium ${getStatusColor(evolution.status)}`}>
                {evolution.status}
              </span>
            </div>
            <div className="bg-gray-50 rounded-lg p-4">
              <p className="text-xs text-gray-500 mb-1">Progress</p>
              <p className="text-lg font-bold text-gray-900">
                {evolution.current_iteration} / {evolution.max_iterations}
              </p>
              <div className="w-full bg-gray-200 rounded-full h-1.5 mt-2">
                <div
                  className="bg-blue-600 h-1.5 rounded-full"
                  style={{ width: `${progress}%` }}
                ></div>
              </div>
              {evolution.status === 'running' && evolution.backtest_progress > 0 && (
                <div className="mt-2">
                  <p className="text-xs text-gray-500">Backtest: {evolution.backtest_progress.toFixed(1)}%</p>
                  <div className="w-full bg-gray-200 rounded-full h-1 mt-1">
                    <div
                      className="bg-green-500 h-1 rounded-full transition-all"
                      style={{ width: `${evolution.backtest_progress}%` }}
                    ></div>
                  </div>
                </div>
              )}
            </div>
            <div className="bg-gray-50 rounded-lg p-4">
              <p className="text-xs text-gray-500 mb-1">Best Version</p>
              <p className="text-lg font-bold text-gray-900">v{evolution.best_version}</p>
            </div>
            <div className="bg-gray-50 rounded-lg p-4">
              <p className="text-xs text-gray-500 mb-1">Best Return</p>
              <div className="flex items-center gap-1">
                {evolution.best_return >= 0 ? (
                  <TrendingUp className="w-4 h-4 text-green-600" />
                ) : (
                  <TrendingDown className="w-4 h-4 text-red-600" />
                )}
                <p className={`text-lg font-bold ${evolution.best_return >= 0 ? 'text-green-600' : 'text-red-600'}`}>
                  {evolution.best_return.toFixed(2)}%
                </p>
              </div>
            </div>
          </div>

          {/* Iteration History */}
          <div>
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Iteration History</h3>
            <IterationTable
              evolutionId={evolution.id}
              onSelectIteration={handleSelectIteration}
            />
          </div>
        </div>

        {/* Iteration Detail Modal */}
        {selectedVersion !== null && (
          <IterationDetailModal
            evolutionId={evolution.id}
            version={selectedVersion}
            isOpen={showIterationDetail}
            onClose={() => setShowIterationDetail(false)}
          />
        )}

        {/* Footer */}
        <div className="flex justify-between p-6 border-t bg-gray-50">
          <div className="flex gap-2">
            {/* Action buttons based on status */}
            {evolution.status === 'created' && (
              <button
                onClick={() => handleAction('start')}
                disabled={loading}
                className="flex items-center gap-2 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50"
              >
                <Play className="w-4 h-4" />
                启动
              </button>
            )}
            {evolution.status === 'running' && (
              <>
                <button
                  onClick={() => handleAction('pause')}
                  disabled={loading}
                  className="flex items-center gap-2 px-4 py-2 bg-yellow-600 text-white rounded-lg hover:bg-yellow-700 disabled:opacity-50"
                >
                  <Pause className="w-4 h-4" />
                  暂停
                </button>
                <button
                  onClick={() => handleAction('stop')}
                  disabled={loading}
                  className="flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
                >
                  <Square className="w-4 h-4" />
                  停止
                </button>
              </>
            )}
            {evolution.status === 'paused' && (
              <>
                <button
                  onClick={() => handleAction('resume')}
                  disabled={loading}
                  className="flex items-center gap-2 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50"
                >
                  <RotateCcw className="w-4 h-4" />
                  恢复
                </button>
                <button
                  onClick={() => handleAction('stop')}
                  disabled={loading}
                  className="flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
                >
                  <Square className="w-4 h-4" />
                  停止
                </button>
              </>
            )}
            {(evolution.status === 'stopped' || evolution.status === 'completed' || evolution.status === 'failed') && (
              <button
                onClick={() => setShowDeleteConfirm(true)}
                disabled={loading}
                className="flex items-center gap-2 px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
              >
                <Trash2 className="w-4 h-4" />
                删除
              </button>
            )}
          </div>
          <button
            onClick={onClose}
            className="px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50"
          >
            关闭
          </button>
        </div>

        {/* Delete Confirmation Modal */}
        {showDeleteConfirm && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-[60]">
            <div className="bg-white rounded-lg p-6 max-w-md">
              <h3 className="text-lg font-semibold mb-4">确认删除</h3>
              <p className="text-gray-600 mb-6">确定要删除此进化任务吗？此操作不可撤销。</p>
              <div className="flex justify-end gap-3">
                <button
                  onClick={() => setShowDeleteConfirm(false)}
                  className="px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50"
                >
                  取消
                </button>
                <button
                  onClick={handleDelete}
                  disabled={loading}
                  className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
                >
                  {loading ? '删除中...' : '确认删除'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
