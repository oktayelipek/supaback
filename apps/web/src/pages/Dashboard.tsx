import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, Job } from '../lib/api'
import { formatDate, formatDuration, formatBytes, formatRelative } from '../lib/utils'
import { StatusBadge, TypeBadge } from '../components/StatusBadge'
import { Modal } from '../components/Modal'

export function Dashboard() {
  const qc = useQueryClient()
  const [triggerOpen, setTriggerOpen] = useState(false)
  const [triggerType, setTriggerType] = useState<'full' | 'database' | 'storage'>('full')

  const { data: jobs = [], isLoading } = useQuery({
    queryKey: ['jobs'],
    queryFn: () => api.listJobs(50),
    refetchInterval: 5000,
  })

  const trigger = useMutation({
    mutationFn: () => api.triggerJob(triggerType),
    onSuccess: () => {
      setTriggerOpen(false)
      setTimeout(() => qc.invalidateQueries({ queryKey: ['jobs'] }), 500)
    },
  })

  const stats = computeStats(jobs)
  const lastJob = jobs[0]

  return (
    <div className="max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-8">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-gray-900">Dashboard</h1>
          <p className="text-sm text-gray-500 mt-0.5">
            {lastJob
              ? `Last backup ${formatRelative(lastJob.StartedAt)}`
              : 'No backups yet'}
          </p>
        </div>
        <button
          onClick={() => setTriggerOpen(true)}
          className="inline-flex items-center gap-2 px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-medium rounded-lg transition-colors shadow-sm"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 0 1 0 1.972l-11.54 6.347a1.125 1.125 0 0 1-1.667-.986V5.653Z" />
          </svg>
          Run Backup
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <StatCard label="Total Jobs" value={stats.total} />
        <StatCard label="Successful" value={stats.success} color="text-emerald-600" />
        <StatCard label="Failed" value={stats.failed} color="text-red-600" />
        <StatCard label="Data Backed Up" value={formatBytes(stats.totalBytes)} />
      </div>

      {/* Jobs table */}
      <div className="bg-white border border-gray-200 rounded-xl overflow-hidden">
        <div className="px-5 py-4 border-b border-gray-100 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-gray-900">Backup History</h2>
          <span className="text-xs text-gray-400">auto-refreshes every 5s</span>
        </div>

        {isLoading ? (
          <div className="py-16 text-center text-sm text-gray-400">Loading…</div>
        ) : jobs.length === 0 ? (
          <EmptyJobs />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-100 bg-gray-50">
                  {['Started', 'Type', 'Status', 'Size', 'Duration', 'Destination'].map(h => (
                    <th key={h} className="px-5 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wide">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-50">
                {jobs.map(job => (
                  <JobRow key={job.ID} job={job} />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Trigger modal */}
      {triggerOpen && (
        <Modal title="Run Backup" onClose={() => setTriggerOpen(false)}>
          <div className="space-y-5">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">Backup type</label>
              <div className="grid grid-cols-3 gap-2">
                {(['full', 'database', 'storage'] as const).map(t => (
                  <button
                    key={t}
                    onClick={() => setTriggerType(t)}
                    className={`py-2.5 rounded-lg border text-sm font-medium transition-colors capitalize ${
                      triggerType === t
                        ? 'border-indigo-500 bg-indigo-50 text-indigo-700'
                        : 'border-gray-200 text-gray-600 hover:border-gray-300'
                    }`}
                  >
                    {t}
                  </button>
                ))}
              </div>
              <p className="mt-2 text-xs text-gray-500">
                {triggerType === 'full' && 'Backs up both database and all storage buckets.'}
                {triggerType === 'database' && 'Backs up PostgreSQL database only (pg_dump).'}
                {triggerType === 'storage' && 'Backs up all Supabase Storage buckets only.'}
              </p>
            </div>
            <div className="flex gap-3 pt-1">
              <button
                onClick={() => setTriggerOpen(false)}
                className="flex-1 py-2 border border-gray-200 rounded-lg text-sm text-gray-600 hover:bg-gray-50 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={() => trigger.mutate()}
                disabled={trigger.isPending}
                className="flex-1 py-2 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium transition-colors"
              >
                {trigger.isPending ? 'Starting…' : 'Start Backup'}
              </button>
            </div>
            {trigger.isError && (
              <p className="text-xs text-red-600">{(trigger.error as Error).message}</p>
            )}
          </div>
        </Modal>
      )}
    </div>
  )
}

function JobRow({ job }: { job: Job }) {
  return (
    <tr className="hover:bg-gray-50 transition-colors">
      <td className="px-5 py-3 whitespace-nowrap text-gray-700">
        <div>{formatDate(job.StartedAt)}</div>
        {job.ErrorMsg && (
          <div className="text-xs text-red-500 mt-0.5 max-w-xs truncate" title={job.ErrorMsg}>
            {job.ErrorMsg}
          </div>
        )}
      </td>
      <td className="px-5 py-3"><TypeBadge type={job.Type} /></td>
      <td className="px-5 py-3"><StatusBadge status={job.Status} /></td>
      <td className="px-5 py-3 text-gray-600 font-mono text-xs">{formatBytes(job.SizeBytes)}</td>
      <td className="px-5 py-3 text-gray-600 font-mono text-xs">{formatDuration(job.StartedAt, job.FinishedAt)}</td>
      <td className="px-5 py-3 text-gray-400 text-xs font-mono truncate max-w-[180px]">{job.Destination}</td>
    </tr>
  )
}

function StatCard({ label, value, color = 'text-gray-900' }: { label: string; value: string | number; color?: string }) {
  return (
    <div className="bg-white border border-gray-200 rounded-xl px-5 py-4">
      <p className="text-xs text-gray-500 font-medium">{label}</p>
      <p className={`text-2xl font-semibold mt-1 ${color}`}>{value}</p>
    </div>
  )
}

function EmptyJobs() {
  return (
    <div className="py-16 text-center">
      <div className="w-12 h-12 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-3">
        <svg className="w-6 h-6 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375" />
        </svg>
      </div>
      <p className="text-sm text-gray-500">No backups yet. Run your first backup above.</p>
    </div>
  )
}

function computeStats(jobs: Job[]) {
  return {
    total: jobs.length,
    success: jobs.filter(j => j.Status === 'success').length,
    failed: jobs.filter(j => j.Status === 'failed').length,
    totalBytes: jobs.filter(j => j.Status === 'success').reduce((s, j) => s + j.SizeBytes, 0),
  }
}
