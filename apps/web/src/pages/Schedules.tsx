import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, Schedule } from '../lib/api'
import { formatDate, formatRelative } from '../lib/utils'
import { TypeBadge } from '../components/StatusBadge'
import { Modal } from '../components/Modal'

const PRESETS = [
  { label: 'Every hour',     value: '0 * * * *' },
  { label: 'Daily 2am',      value: '0 2 * * *' },
  { label: 'Daily midnight', value: '0 0 * * *' },
  { label: 'Weekly Sun 2am', value: '0 2 * * 0' },
  { label: 'Monthly 1st',    value: '0 2 1 * *' },
]

export function Schedules() {
  const qc = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)

  const { data: schedules = [], isLoading } = useQuery({
    queryKey: ['schedules'],
    queryFn: api.listSchedules,
  })

  const toggle = useMutation({
    mutationFn: ({ id, enabled }: { id: number; enabled: boolean }) =>
      api.toggleSchedule(id, enabled),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['schedules'] }),
  })

  const remove = useMutation({
    mutationFn: (id: number) => api.deleteSchedule(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['schedules'] }),
  })

  return (
    <div className="max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-gray-900">Schedules</h1>
          <p className="text-sm text-gray-500 mt-0.5">Automate backups with cron expressions</p>
        </div>
        <button
          onClick={() => setCreateOpen(true)}
          className="inline-flex items-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium rounded-lg transition-colors shadow-sm"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
          </svg>
          Add Schedule
        </button>
      </div>

      {isLoading ? (
        <div className="py-16 text-center text-sm text-gray-400">Loading…</div>
      ) : schedules.length === 0 ? (
        <EmptySchedules onAdd={() => setCreateOpen(true)} />
      ) : (
        <div className="space-y-3">
          {schedules.map(sc => (
            <ScheduleRow
              key={sc.ID}
              schedule={sc}
              onToggle={(enabled) => toggle.mutate({ id: sc.ID, enabled })}
              onDelete={() => {
                if (confirm(`Delete schedule "${sc.Name}"?`)) {
                  remove.mutate(sc.ID)
                }
              }}
            />
          ))}
        </div>
      )}

      {createOpen && (
        <CreateModal
          onClose={() => setCreateOpen(false)}
          onCreated={() => {
            setCreateOpen(false)
            qc.invalidateQueries({ queryKey: ['schedules'] })
          }}
        />
      )}
    </div>
  )
}

function ScheduleRow({
  schedule,
  onToggle,
  onDelete,
}: {
  schedule: Schedule
  onToggle: (enabled: boolean) => void
  onDelete: () => void
}) {
  return (
    <div className={`bg-white border rounded-xl px-5 py-4 flex items-center gap-4 transition-opacity ${
      schedule.Enabled ? 'border-gray-200' : 'border-gray-100 opacity-60'
    }`}>
      <div className="flex-1 min-w-0 space-y-1.5">
        <div className="flex items-center gap-2">
          <span className="font-medium text-sm text-gray-900">{schedule.Name}</span>
          <TypeBadge type={schedule.Type} />
        </div>
        <code className="text-xs font-mono text-indigo-600 bg-indigo-50 px-2 py-0.5 rounded">
          {schedule.CronExpr}
        </code>
        <p className="text-xs text-gray-400">
          {schedule.LastRunAt
            ? `Last run ${formatRelative(schedule.LastRunAt)} · ${formatDate(schedule.LastRunAt)}`
            : 'Never run'}
        </p>
      </div>

      <div className="flex items-center gap-3 shrink-0">
        {/* Toggle */}
        <button
          onClick={() => onToggle(!schedule.Enabled)}
          className={`relative w-10 h-5 rounded-full transition-colors focus:outline-none ${
            schedule.Enabled ? 'bg-indigo-600' : 'bg-gray-200'
          }`}
          title={schedule.Enabled ? 'Disable' : 'Enable'}
        >
          <span className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${
            schedule.Enabled ? 'translate-x-5' : 'translate-x-0'
          }`} />
        </button>

        {/* Delete */}
        <button
          onClick={onDelete}
          className="p-1.5 text-gray-300 hover:text-red-500 transition-colors rounded"
          title="Delete schedule"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916" />
          </svg>
        </button>
      </div>
    </div>
  )
}

function CreateModal({ onClose, onCreated }: { onClose: () => void; onCreated: () => void }) {
  const [name, setName] = useState('')
  const [cronExpr, setCronExpr] = useState('0 2 * * *')
  const [type, setType] = useState<'full' | 'database' | 'storage'>('full')
  const [error, setError] = useState('')

  const create = useMutation({
    mutationFn: () => api.createSchedule({ name, cron_expr: cronExpr, type }),
    onSuccess: onCreated,
    onError: (e: Error) => setError(e.message),
  })

  return (
    <Modal title="Add Schedule" onClose={onClose}>
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
          <input
            type="text"
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="Daily backup"
            className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Backup Type</label>
          <div className="grid grid-cols-3 gap-2">
            {(['full', 'database', 'storage'] as const).map(t => (
              <button
                key={t}
                onClick={() => setType(t)}
                className={`py-2 rounded-lg border text-xs font-medium transition-colors capitalize ${
                  type === t
                    ? 'border-indigo-500 bg-indigo-50 text-indigo-700'
                    : 'border-gray-200 text-gray-600 hover:border-gray-300'
                }`}
              >
                {t}
              </button>
            ))}
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Cron Expression</label>
          <input
            type="text"
            value={cronExpr}
            onChange={e => setCronExpr(e.target.value)}
            className="w-full px-3 py-2 border border-gray-200 rounded-lg text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
          />
          <div className="mt-2 flex flex-wrap gap-1.5">
            {PRESETS.map(p => (
              <button
                key={p.value}
                onClick={() => setCronExpr(p.value)}
                className={`px-2 py-0.5 rounded text-xs border transition-colors ${
                  cronExpr === p.value
                    ? 'border-indigo-400 bg-indigo-50 text-indigo-700'
                    : 'border-gray-200 text-gray-500 hover:border-gray-300'
                }`}
              >
                {p.label}
              </button>
            ))}
          </div>
        </div>

        {error && <p className="text-xs text-red-600">{error}</p>}

        <div className="flex gap-3 pt-1">
          <button
            onClick={onClose}
            className="flex-1 py-2 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={() => create.mutate()}
            disabled={create.isPending || !name.trim() || !cronExpr.trim()}
            className="flex-1 py-2 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium transition-colors"
          >
            {create.isPending ? 'Creating…' : 'Create'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

function EmptySchedules({ onAdd }: { onAdd: () => void }) {
  return (
    <div className="py-16 text-center bg-white border border-gray-200 rounded-xl">
      <div className="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-3">
        <svg className="w-6 h-6 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
        </svg>
      </div>
      <p className="text-sm text-gray-500 mb-4">No schedules yet.</p>
      <button
        onClick={onAdd}
        className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium rounded-lg transition-colors"
      >
        Add your first schedule
      </button>
    </div>
  )
}
