import { useState, useEffect } from 'react'
import { Plus, RefreshCw } from 'lucide-react'
import { api } from '../lib/api'
import type { Evolution } from '../types'
import EvolutionCard from '../components/evolution/EvolutionCard'
import CreateEvolutionModal from '../components/evolution/CreateEvolutionModal'
import EvolutionDetailModal from '../components/evolution/EvolutionDetailModal'

export default function EvolutionPage() {
  const [evolutions, setEvolutions] = useState<Evolution[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedEvolution, setSelectedEvolution] = useState<Evolution | null>(null)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showDetailModal, setShowDetailModal] = useState(false)

  // Load evolutions list
  useEffect(() => {
    loadEvolutions()
  }, [])

  const loadEvolutions = async () => {
    try {
      setLoading(true)
      const data = await api.listEvolutions()
      setEvolutions(data)
    } catch (error) {
      console.error('Failed to load evolutions:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleStart = async (id: string) => {
    try {
      await api.startEvolution(id)
      await loadEvolutions()
    } catch (error) {
      console.error('Failed to start evolution:', error)
    }
  }

  const handlePause = async (id: string) => {
    try {
      await api.pauseEvolution(id)
      await loadEvolutions()
    } catch (error) {
      console.error('Failed to pause evolution:', error)
    }
  }

  const handleStop = async (id: string) => {
    try {
      await api.stopEvolution(id)
      await loadEvolutions()
    } catch (error) {
      console.error('Failed to stop evolution:', error)
    }
  }

  const handleSelectEvolution = (evolution: Evolution) => {
    setSelectedEvolution(evolution)
    setShowDetailModal(true)
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">🧬 Evolution Lab</h1>
          <p className="text-sm text-gray-500 mt-1">
            AI-driven strategy evolution and optimization
          </p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={loadEvolutions}
            className="px-4 py-2 text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 flex items-center gap-2"
          >
            <RefreshCw className="w-4 h-4" />
            Refresh
          </button>
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 flex items-center gap-2"
          >
            <Plus className="w-4 h-4" />
            New Evolution
          </button>
        </div>
      </div>

      {/* Content */}
      {loading ? (
        <div className="text-center py-12">
          <div
            className="inline-block animate-spin rounded-full h-8 w-8 border-b-2"
            style={{ borderColor: '#3B82F6' }}
          ></div>
          <p className="mt-2" style={{ color: '#848E9C' }}>Loading...</p>
        </div>
      ) : evolutions.length === 0 ? (
        <div
          className="text-center py-12 rounded-lg border-2 border-dashed"
          style={{
            background: 'rgba(59, 130, 246, 0.05)',
            borderColor: 'rgba(59, 130, 246, 0.2)'
          }}
        >
          <div className="text-6xl mb-4">🧬</div>
          <p className="text-lg font-semibold mb-2" style={{ color: '#EAECEF' }}>
            No evolution tasks yet
          </p>
          <p className="text-sm mb-4" style={{ color: '#848E9C' }}>
            Create your first evolution task to start AI-driven strategy optimization
          </p>
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-6 py-3 rounded-lg hover:opacity-90 transition-opacity font-semibold"
            style={{
              background: 'linear-gradient(135deg, #3B82F6 0%, #2563EB 100%)',
              color: '#FFFFFF'
            }}
          >
            Create First Evolution
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {evolutions.map((evolution) => (
            <EvolutionCard
              key={evolution.id}
              evolution={evolution}
              onStart={handleStart}
              onPause={handlePause}
              onStop={handleStop}
              onSelect={handleSelectEvolution}
            />
          ))}
        </div>
      )}

      {/* Create Modal */}
      <CreateEvolutionModal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSuccess={loadEvolutions}
      />

      {/* Detail Modal */}
      <EvolutionDetailModal
        evolution={selectedEvolution}
        isOpen={showDetailModal}
        onClose={() => setShowDetailModal(false)}
        onRefresh={loadEvolutions}
      />
    </div>
  )
}
