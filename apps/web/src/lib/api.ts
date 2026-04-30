export interface Settings {
  configured: boolean
  supabase_url: string
  supabase_service_key: string
  supabase_db_url: string
  backup_include_database: string
  backup_include_storage: string
  backup_compress: string
  backup_buckets: string
  destination_type: string
  destination_local_path: string
  destination_s3_endpoint: string
  destination_s3_region: string
  destination_s3_bucket: string
  destination_s3_prefix: string
  destination_s3_access_key_id: string
  destination_s3_secret_access_key: string
  destination_s3_force_path_style: string
}

export interface Job {
  ID: number
  StartedAt: string
  FinishedAt: string | null
  Status: 'running' | 'success' | 'failed'
  Type: 'full' | 'database' | 'storage'
  Destination: string
  SizeBytes: number
  ErrorMsg: string
}

export interface Schedule {
  ID: number
  Name: string
  CronExpr: string
  Type: 'full' | 'database' | 'storage'
  Enabled: boolean
  CreatedAt: string
  LastRunAt: string | null
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error ?? res.statusText)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

export const api = {
  getSettings: () =>
    request<Settings>('/api/settings'),

  updateSettings: (data: Partial<Record<string, string>>) =>
    request<{ status: string; configured: boolean }>('/api/settings', {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  listJobs: (limit = 50) =>
    request<Job[]>(`/api/jobs?limit=${limit}`),

  triggerJob: (type: 'full' | 'database' | 'storage') =>
    request<{ status: string; type: string }>('/api/jobs', {
      method: 'POST',
      body: JSON.stringify({ type }),
    }),

  listSchedules: () =>
    request<Schedule[]>('/api/schedules'),

  createSchedule: (payload: { name: string; cron_expr: string; type: string }) =>
    request<Schedule>('/api/schedules', {
      method: 'POST',
      body: JSON.stringify(payload),
    }),

  toggleSchedule: (id: number, enabled: boolean) =>
    request<Schedule>(`/api/schedules/${id}/toggle`, {
      method: 'PATCH',
      body: JSON.stringify({ enabled }),
    }),

  deleteSchedule: (id: number) =>
    request<void>(`/api/schedules/${id}`, { method: 'DELETE' }),
}
