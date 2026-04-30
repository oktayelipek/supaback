import { cx } from '../lib/utils'

const styles = {
  running: 'bg-amber-100 text-amber-800 border-amber-200',
  success: 'bg-emerald-100 text-emerald-800 border-emerald-200',
  failed:  'bg-red-100 text-red-800 border-red-200',
} as const

const labels = {
  running: '● Running',
  success: '✓ Success',
  failed:  '✕ Failed',
} as const

type Status = keyof typeof styles

export function StatusBadge({ status }: { status: Status }) {
  return (
    <span className={cx(
      'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium border',
      styles[status]
    )}>
      {labels[status]}
    </span>
  )
}

const typeStyles = {
  full:     'bg-indigo-100 text-indigo-800 border-indigo-200',
  database: 'bg-blue-100 text-blue-800 border-blue-200',
  storage:  'bg-purple-100 text-purple-800 border-purple-200',
} as const

type JobType = keyof typeof typeStyles

export function TypeBadge({ type }: { type: JobType }) {
  return (
    <span className={cx(
      'inline-flex items-center px-2 py-0.5 rounded text-xs font-medium border',
      typeStyles[type]
    )}>
      {type}
    </span>
  )
}
