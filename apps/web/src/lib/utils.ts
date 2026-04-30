import { formatDistanceToNow, format, differenceInSeconds } from 'date-fns'

export function formatDate(iso: string) {
  return format(new Date(iso), 'MMM d, yyyy HH:mm')
}

export function formatRelative(iso: string) {
  return formatDistanceToNow(new Date(iso), { addSuffix: true })
}

export function formatDuration(start: string, end: string | null) {
  if (!end) return '—'
  const secs = differenceInSeconds(new Date(end), new Date(start))
  if (secs < 60) return `${secs}s`
  const mins = Math.floor(secs / 60)
  const rem = secs % 60
  return rem > 0 ? `${mins}m ${rem}s` : `${mins}m`
}

export function formatBytes(bytes: number) {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

export function cx(...args: (string | undefined | false | null)[]) {
  return args.filter(Boolean).join(' ')
}
