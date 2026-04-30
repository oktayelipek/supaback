import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api, BackupFile } from '../lib/api'
import { formatBytes } from '../lib/utils'

export function Backups() {
  const { data: dates = [], isLoading } = useQuery({
    queryKey: ['backups'],
    queryFn: api.listBackups,
  })

  if (isLoading) {
    return <div className="max-w-4xl mx-auto px-4 py-8 text-sm text-gray-400">Loading…</div>
  }

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 py-8 space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-gray-900">Backups</h1>
        <p className="text-sm text-gray-500 mt-0.5">Browse and download your backup files</p>
      </div>

      {dates.length === 0 ? (
        <div className="bg-white border border-gray-200 rounded-xl px-6 py-12 text-center">
          <p className="text-sm text-gray-500">No backups yet. Run a backup from the Dashboard.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {dates.map(d => (
            <DateCard key={d.date} date={d.date} totalBytes={d.total_bytes} files={d.files} />
          ))}
        </div>
      )}
    </div>
  )
}

function DateCard({ date, totalBytes, files }: { date: string; totalBytes: number; files: BackupFile[] }) {
  const [open, setOpen] = useState(false)

  const dbFiles = files.filter(f => f.key.includes('/database/'))
  const storageFiles = files.filter(f => f.key.includes('/storage/'))

  return (
    <div className="bg-white border border-gray-200 rounded-xl overflow-hidden">
      {/* Header row */}
      <div className="flex items-center justify-between px-5 py-4">
        <button
          onClick={() => setOpen(o => !o)}
          className="flex items-center gap-3 flex-1 text-left"
        >
          <svg className={`w-4 h-4 text-gray-400 transition-transform ${open ? 'rotate-90' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="m8.25 4.5 7.5 7.5-7.5 7.5" />
          </svg>
          <span className="text-sm font-semibold text-gray-900">{date}</span>
          <div className="flex items-center gap-2">
            {dbFiles.length > 0 && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-indigo-50 text-indigo-700 text-xs font-medium">
                <DatabaseIcon /> DB
              </span>
            )}
            {storageFiles.length > 0 && (
              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-emerald-50 text-emerald-700 text-xs font-medium">
                <FolderIcon /> Storage
              </span>
            )}
          </div>
        </button>
        <div className="flex items-center gap-3">
          <span className="text-xs text-gray-400">{files.length} file{files.length !== 1 ? 's' : ''}</span>
          <span className="text-xs font-medium text-gray-600">{formatBytes(totalBytes)}</span>
          <button
            onClick={() => api.downloadBackupDate(date)}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-indigo-600 hover:text-indigo-700 hover:bg-indigo-50 border border-indigo-200 rounded-lg transition-colors"
            title={`Download all files for ${date} as zip`}
          >
            <DownloadIcon />
            Download all
          </button>
        </div>
      </div>

      {/* File list */}
      {open && (
        <div className="border-t border-gray-100 divide-y divide-gray-50">
          {files.map(f => (
            <FileRow key={f.key} file={f} />
          ))}
        </div>
      )}
    </div>
  )
}

function FileRow({ file }: { file: BackupFile }) {
  const isDb = file.key.includes('/database/')
  return (
    <div className="flex items-center justify-between px-5 py-3 hover:bg-gray-50 group">
      <div className="flex items-center gap-3 min-w-0">
        <span className={`shrink-0 ${isDb ? 'text-indigo-400' : 'text-emerald-400'}`}>
          {isDb ? <DatabaseIcon /> : <FileIcon />}
        </span>
        <span className="text-sm text-gray-700 truncate">{file.name}</span>
        <span className="text-xs text-gray-400 shrink-0">{formatBytes(file.size_bytes)}</span>
      </div>
      <button
        onClick={() => api.downloadBackup(file.key)}
        className="shrink-0 ml-4 flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-indigo-600 hover:text-indigo-700 hover:bg-indigo-50 rounded-lg transition-colors opacity-0 group-hover:opacity-100"
      >
        <DownloadIcon />
        Download
      </button>
    </div>
  )
}

function DatabaseIcon() {
  return (
    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 5.625c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
    </svg>
  )
}

function FolderIcon() {
  return (
    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 12.75V12A2.25 2.25 0 0 1 4.5 9.75h15A2.25 2.25 0 0 1 21.75 12v.75m-8.69-6.44-2.12-2.12a1.5 1.5 0 0 0-1.061-.44H4.5A2.25 2.25 0 0 0 2.25 6v12a2.25 2.25 0 0 0 2.25 2.25h15A2.25 2.25 0 0 0 21.75 18V9a2.25 2.25 0 0 0-2.25-2.25h-5.379a1.5 1.5 0 0 1-1.06-.44Z" />
    </svg>
  )
}

function FileIcon() {
  return (
    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />
    </svg>
  )
}

function DownloadIcon() {
  return (
    <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3" />
    </svg>
  )
}
