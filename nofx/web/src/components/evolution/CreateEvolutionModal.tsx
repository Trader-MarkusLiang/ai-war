import { useState, useEffect } from 'react'
import { X } from 'lucide-react'
import { api } from '../../lib/api'
import type { Strategy, CreateEvolutionRequest, AIModel } from '../../types'

const TIMEFRAME_OPTIONS = ['1m', '3m', '5m', '15m', '30m', '1h', '4h', '1d']

interface CreateEvolutionModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: () => void
}

export default function CreateEvolutionModal({
  isOpen,
  onClose,
  onSuccess,
}: CreateEvolutionModalProps) {
  const [strategies, setStrategies] = useState<Strategy[]>([])
  const [aiModels, setAIModels] = useState<AIModel[]>([])
  const [loading, setLoading] = useState(false)

  const [formData, setFormData] = useState({
    name: '',
    base_strategy_id: '',
    max_iterations: 10,
    convergence_threshold: 3,
    symbols: 'BTC,ETH',
    timeframes: '1h,4h',
    start_date: '',
    end_date: '',
    initial_balance: 10000,
    fee_bps: 5,
    slippage_bps: 2,
    decision_timeframe: '4h',
    decision_cadence: 1,
    btc_eth_leverage: 5,
    altcoin_leverage: 5,
    ai_model_id: '',
    cache_ai: true,
  })

  useEffect(() => {
    if (isOpen) {
      loadStrategies()
      loadAIModels()
      // Set default dates (last 30 days)
      const end = new Date()
      const start = new Date()
      start.setDate(start.getDate() - 30)
      setFormData(prev => ({
        ...prev,
        start_date: start.toISOString().split('T')[0],
        end_date: end.toISOString().split('T')[0],
      }))
    }
  }, [isOpen])

  const loadStrategies = async () => {
    try {
      const data = await api.getStrategies()
      setStrategies(data)
    } catch (error) {
      console.error('Failed to load strategies:', error)
    }
  }

  const loadAIModels = async () => {
    try {
      const data = await api.getModelConfigs()
      setAIModels(data)
      if (data.length > 0 && !formData.ai_model_id) {
        setFormData(prev => ({ ...prev, ai_model_id: data[0].id }))
      }
    } catch (error) {
      console.error('Failed to load AI models:', error)
    }
  }

  // Normalize symbol names to full trading pairs
  const normalizeSymbols = (input: string): string[] => {
    return input.split(',').map(s => {
      const trimmed = s.trim().toUpperCase()
      // If already a full pair (contains USDT, BUSD, etc.), use as-is
      if (trimmed.includes('USDT') || trimmed.includes('BUSD') || trimmed.includes('USDC')) {
        return trimmed
      }
      // Otherwise, append USDT
      return `${trimmed}USDT`
    })
  }

  const handleSubmit = async () => {
    if (!formData.name || !formData.base_strategy_id) {
      alert('Please fill in required fields')
      return
    }

    try {
      setLoading(true)
      const request: CreateEvolutionRequest = {
        name: formData.name,
        base_strategy_id: formData.base_strategy_id,
        max_iterations: formData.max_iterations,
        convergence_threshold: formData.convergence_threshold,
        fixed_params: {
          symbols: normalizeSymbols(formData.symbols),
          timeframes: formData.timeframes.split(',').map(s => s.trim()),
          start_ts: new Date(formData.start_date).getTime() / 1000,
          end_ts: new Date(formData.end_date).getTime() / 1000,
          initial_balance: formData.initial_balance,
          fee_bps: formData.fee_bps,
          slippage_bps: formData.slippage_bps,
          decision_timeframe: formData.decision_timeframe,
          decision_cadence_nbars: formData.decision_cadence,
          btc_eth_leverage: formData.btc_eth_leverage,
          altcoin_leverage: formData.altcoin_leverage,
          ai_model_id: formData.ai_model_id,
          cache_ai: formData.cache_ai,
        },
      }

      await api.createEvolution(request)
      onSuccess()
      onClose()
    } catch (error) {
      console.error('Failed to create evolution:', error)
      alert('Failed to create evolution task')
    } finally {
      setLoading(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black bg-opacity-70 flex items-center justify-center z-50 p-4">
      <div
        className="rounded-lg shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-y-auto"
        style={{
          background: '#1E2329',
          border: '1px solid #2B3139'
        }}
      >
        {/* Header */}
        <div
          className="flex items-center justify-between p-6 border-b"
          style={{ borderColor: '#2B3139' }}
        >
          <div>
            <h2 className="text-xl font-bold" style={{ color: '#EAECEF' }}>
              🧬 Create Evolution Task
            </h2>
            <p className="text-sm mt-1" style={{ color: '#848E9C' }}>
              AI-driven strategy evolution and optimization
            </p>
          </div>
          <button
            onClick={onClose}
            className="p-2 rounded-lg transition-colors hover:bg-gray-700"
            style={{ color: '#848E9C' }}
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Form */}
        <div className="p-6 space-y-4">
          {/* Basic Info */}
          <div>
            <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
              Task Name *
            </label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
              style={{
                background: '#2B3139',
                border: '1px solid #3C4043',
                color: '#EAECEF'
              }}
              placeholder="e.g., BTC Strategy Evolution v1"
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
              Base Strategy *
            </label>
            <select
              value={formData.base_strategy_id}
              onChange={(e) => setFormData({ ...formData, base_strategy_id: e.target.value })}
              className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
              style={{
                background: '#2B3139',
                border: '1px solid #3C4043',
                color: '#EAECEF'
              }}
            >
              <option value="">Select a strategy</option>
              {strategies.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </div>

          {/* Evolution Settings */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
                Max Iterations
              </label>
              <input
                type="number"
                value={formData.max_iterations}
                onChange={(e) => setFormData({ ...formData, max_iterations: parseInt(e.target.value) })}
                className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
                style={{
                  background: '#2B3139',
                  border: '1px solid #3C4043',
                  color: '#EAECEF'
                }}
                min="1"
                max="50"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
                Convergence Threshold
              </label>
              <input
                type="number"
                value={formData.convergence_threshold}
                onChange={(e) => setFormData({ ...formData, convergence_threshold: parseInt(e.target.value) })}
                className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
                style={{
                  background: '#2B3139',
                  border: '1px solid #3C4043',
                  color: '#EAECEF'
                }}
                min="1"
                max="10"
              />
            </div>
          </div>

          {/* Backtest Parameters */}
          <div className="border-t pt-4" style={{ borderColor: '#2B3139' }}>
            <h3 className="text-sm font-semibold mb-3" style={{ color: '#EAECEF' }}>
              Backtest Parameters
            </h3>

            <div className="space-y-3">
              <div>
                <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
                  Symbols (comma-separated)
                </label>
                <input
                  type="text"
                  value={formData.symbols}
                  onChange={(e) => setFormData({ ...formData, symbols: e.target.value })}
                  className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
                  style={{
                    background: '#2B3139',
                    border: '1px solid #3C4043',
                    color: '#EAECEF'
                  }}
                  placeholder="BTC,SOL,DOGE (auto-converts to USDT pairs)"
                />
              </div>

              <div>
                <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
                  Timeframes (comma-separated)
                </label>
                <input
                  type="text"
                  value={formData.timeframes}
                  onChange={(e) => setFormData({ ...formData, timeframes: e.target.value })}
                  className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
                  style={{
                    background: '#2B3139',
                    border: '1px solid #3C4043',
                    color: '#EAECEF'
                  }}
                  placeholder="1h,4h"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
                    Start Date
                  </label>
                  <input
                    type="date"
                    value={formData.start_date}
                    onChange={(e) => setFormData({ ...formData, start_date: e.target.value })}
                    className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
                    style={{
                      background: '#2B3139',
                      border: '1px solid #3C4043',
                      color: '#EAECEF'
                    }}
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
                    End Date
                  </label>
                  <input
                    type="date"
                    value={formData.end_date}
                    onChange={(e) => setFormData({ ...formData, end_date: e.target.value })}
                    className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
                    style={{
                      background: '#2B3139',
                      border: '1px solid #3C4043',
                      color: '#EAECEF'
                    }}
                  />
                </div>
              </div>
            </div>
          </div>

          {/* AI Model Selection */}
          <div className="border-t pt-4" style={{ borderColor: '#2B3139' }}>
            <h3 className="text-sm font-semibold mb-3" style={{ color: '#EAECEF' }}>
              AI Model
            </h3>
            <select
              value={formData.ai_model_id}
              onChange={(e) => setFormData({ ...formData, ai_model_id: e.target.value })}
              className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
              style={{
                background: '#2B3139',
                border: '1px solid #3C4043',
                color: '#EAECEF'
              }}
            >
              <option value="">Select AI Model</option>
              {aiModels.filter(m => m.enabled).map((model) => (
                <option key={model.id} value={model.id}>
                  {model.displayName || model.name} ({model.provider})
                </option>
              ))}
            </select>
          </div>

          {/* Decision Settings */}
          <div className="border-t pt-4" style={{ borderColor: '#2B3139' }}>
            <h3 className="text-sm font-semibold mb-3" style={{ color: '#EAECEF' }}>
              Decision Settings
            </h3>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
                  Decision Timeframe
                </label>
                <select
                  value={formData.decision_timeframe}
                  onChange={(e) => setFormData({ ...formData, decision_timeframe: e.target.value })}
                  className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
                  style={{
                    background: '#2B3139',
                    border: '1px solid #3C4043',
                    color: '#EAECEF'
                  }}
                >
                  {TIMEFRAME_OPTIONS.map((tf) => (
                    <option key={tf} value={tf}>{tf}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
                  Decision Cadence (bars)
                </label>
                <input
                  type="number"
                  value={formData.decision_cadence}
                  onChange={(e) => setFormData({ ...formData, decision_cadence: parseInt(e.target.value) || 1 })}
                  className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
                  style={{
                    background: '#2B3139',
                    border: '1px solid #3C4043',
                    color: '#EAECEF'
                  }}
                  min="1"
                  max="24"
                />
              </div>
            </div>
          </div>

          {/* Leverage Settings */}
          <div className="border-t pt-4" style={{ borderColor: '#2B3139' }}>
            <h3 className="text-sm font-semibold mb-3" style={{ color: '#EAECEF' }}>
              Leverage Settings
            </h3>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
                  BTC/ETH Leverage
                </label>
                <input
                  type="number"
                  value={formData.btc_eth_leverage}
                  onChange={(e) => setFormData({ ...formData, btc_eth_leverage: parseInt(e.target.value) || 1 })}
                  className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
                  style={{
                    background: '#2B3139',
                    border: '1px solid #3C4043',
                    color: '#EAECEF'
                  }}
                  min="1"
                  max="125"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
                  Altcoin Leverage
                </label>
                <input
                  type="number"
                  value={formData.altcoin_leverage}
                  onChange={(e) => setFormData({ ...formData, altcoin_leverage: parseInt(e.target.value) || 1 })}
                  className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
                  style={{
                    background: '#2B3139',
                    border: '1px solid #3C4043',
                    color: '#EAECEF'
                  }}
                  min="1"
                  max="75"
                />
              </div>
            </div>
          </div>

          {/* Trading Settings */}
          <div className="border-t pt-4" style={{ borderColor: '#2B3139' }}>
            <h3 className="text-sm font-semibold mb-3" style={{ color: '#EAECEF' }}>
              Trading Settings
            </h3>
            <div className="grid grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
                  Initial Balance (USDT)
                </label>
                <input
                  type="number"
                  value={formData.initial_balance}
                  onChange={(e) => setFormData({ ...formData, initial_balance: parseFloat(e.target.value) || 10000 })}
                  className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
                  style={{
                    background: '#2B3139',
                    border: '1px solid #3C4043',
                    color: '#EAECEF'
                  }}
                  min="100"
                  step="100"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
                  Fee (bps)
                </label>
                <input
                  type="number"
                  value={formData.fee_bps}
                  onChange={(e) => setFormData({ ...formData, fee_bps: parseFloat(e.target.value) || 0 })}
                  className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
                  style={{
                    background: '#2B3139',
                    border: '1px solid #3C4043',
                    color: '#EAECEF'
                  }}
                  min="0"
                  max="100"
                  step="0.5"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1" style={{ color: '#EAECEF' }}>
                  Slippage (bps)
                </label>
                <input
                  type="number"
                  value={formData.slippage_bps}
                  onChange={(e) => setFormData({ ...formData, slippage_bps: parseFloat(e.target.value) || 0 })}
                  className="w-full px-3 py-2 rounded-lg focus:ring-2 focus:ring-blue-500 focus:outline-none"
                  style={{
                    background: '#2B3139',
                    border: '1px solid #3C4043',
                    color: '#EAECEF'
                  }}
                  min="0"
                  max="100"
                  step="0.5"
                />
              </div>
            </div>

            {/* Cache AI Option */}
            <div className="mt-4 flex items-center gap-2">
              <input
                type="checkbox"
                id="cache_ai"
                checked={formData.cache_ai}
                onChange={(e) => setFormData({ ...formData, cache_ai: e.target.checked })}
                className="w-4 h-4 rounded"
              />
              <label htmlFor="cache_ai" className="text-sm" style={{ color: '#EAECEF' }}>
                Cache AI Responses (faster iterations, lower cost)
              </label>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div
          className="flex justify-end gap-3 p-6 border-t"
          style={{
            borderColor: '#2B3139',
            background: '#181A20'
          }}
        >
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-lg hover:bg-gray-700 transition-colors"
            style={{
              color: '#EAECEF',
              background: '#2B3139',
              border: '1px solid #3C4043'
            }}
            disabled={loading}
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            className="px-4 py-2 rounded-lg hover:opacity-90 transition-opacity disabled:opacity-50"
            style={{
              background: 'linear-gradient(135deg, #3B82F6 0%, #2563EB 100%)',
              color: '#FFFFFF'
            }}
            disabled={loading}
          >
            {loading ? 'Creating...' : 'Create Evolution'}
          </button>
        </div>
      </div>
    </div>
  )
}
