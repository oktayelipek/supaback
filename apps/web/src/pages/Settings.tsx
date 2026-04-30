import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'

export function Settings_() {
  const qc = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: ['settings'], queryFn: api.getSettings })
  const [form, setForm] = useState<Partial<Record<string, string>>>({})
  const [saved, setSaved] = useState(false)
  const [destType, setDestType] = useState('local')

  useEffect(() => {
    if (data) {
      setForm(data as unknown as Record<string, string>)
      setDestType(data.destination_type || 'local')
    }
  }, [data])

  const save = useMutation({
    mutationFn: () => api.updateSettings(form),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['settings'] })
      qc.invalidateQueries({ queryKey: ['health'] })
      setSaved(true)
      setTimeout(() => setSaved(false), 3000)
    },
  })

  const set = (key: string, value: string) => {
    setForm(f => ({ ...f, [key]: value }))
    if (key === 'destination_type') setDestType(value)
  }

  if (isLoading) {
    return <div className="max-w-2xl mx-auto px-4 py-8 text-sm text-gray-400">Loading…</div>
  }

  return (
    <div className="max-w-2xl mx-auto px-4 sm:px-6 py-8 space-y-8">
      <div>
        <h1 className="text-xl font-semibold text-gray-900">Settings</h1>
        <p className="text-sm text-gray-500 mt-0.5">Configure your Supabase connection and backup destination</p>
      </div>

      {/* Supabase */}
      <Section title="Supabase" description="Your project credentials from the Supabase dashboard → Settings → API">
        <Field label="Project URL" required>
          <Input
            type="url"
            placeholder="https://xxxxxxxxxxxx.supabase.co"
            value={form['supabase_url'] ?? ''}
            onChange={e => set('supabase_url', e.target.value)}
          />
        </Field>
        <Field label="Service Role Key" required hint="Settings → API → service_role (keep this secret)">
          <Input
            type="password"
            placeholder="eyJhbGci…"
            value={form['supabase_service_key'] ?? ''}
            onChange={e => set('supabase_service_key', e.target.value)}
          />
        </Field>
        <Field label="Database URL" hint="Settings → Database → Connection string (URI format)">
          <Input
            type="password"
            placeholder="postgresql://postgres:[password]@db.xxxx.supabase.co:5432/postgres"
            value={form['supabase_db_url'] ?? ''}
            onChange={e => set('supabase_db_url', e.target.value)}
          />
          <p className="mt-1 text-xs text-gray-400">Required only if database backup is enabled</p>
        </Field>
      </Section>

      {/* Backup Options */}
      <Section title="Backup Options">
        <div className="space-y-3">
          <Toggle
            label="Database backup"
            description="Run pg_dump on your PostgreSQL database"
            checked={form['backup_include_database'] !== 'false'}
            onChange={v => set('backup_include_database', String(v))}
          />
          <Toggle
            label="Storage backup"
            description="Download all files from Supabase Storage buckets"
            checked={form['backup_include_storage'] !== 'false'}
            onChange={v => set('backup_include_storage', String(v))}
          />
          <Toggle
            label="Compress backups"
            description="Gzip database dumps to reduce file size"
            checked={form['backup_compress'] !== 'false'}
            onChange={v => set('backup_compress', String(v))}
          />
        </div>
        <Field label="Buckets" hint="Comma-separated list. Leave empty to back up all buckets.">
          <Input
            placeholder="avatars, documents, assets"
            value={form['backup_buckets'] ?? ''}
            onChange={e => set('backup_buckets', e.target.value)}
          />
        </Field>
      </Section>

      {/* Destination */}
      <Section title="Destination" description="Where to store your backup files">
        <Field label="Storage type">
          <div className="grid grid-cols-2 gap-2">
            {[
              { value: 'local', label: 'Local', sub: 'Save to disk' },
              { value: 's3', label: 'S3-compatible', sub: 'AWS S3, R2, MinIO' },
            ].map(opt => (
              <button
                key={opt.value}
                onClick={() => set('destination_type', opt.value)}
                className={`p-3 rounded-lg border text-left transition-colors ${
                  destType === opt.value
                    ? 'border-indigo-500 bg-indigo-50'
                    : 'border-gray-200 hover:border-gray-300'
                }`}
              >
                <div className={`text-sm font-medium ${destType === opt.value ? 'text-indigo-700' : 'text-gray-700'}`}>
                  {opt.label}
                </div>
                <div className="text-xs text-gray-400 mt-0.5">{opt.sub}</div>
              </button>
            ))}
          </div>
        </Field>

        {destType === 'local' && (
          <Field label="Local path">
            <Input
              placeholder="./backups"
              value={form['destination_local_path'] ?? ''}
              onChange={e => set('destination_local_path', e.target.value)}
            />
          </Field>
        )}

        {destType === 's3' && (
          <>
            <Field label="Endpoint" hint="Leave empty for AWS S3. For R2: https://ACCOUNT.r2.cloudflarestorage.com">
              <Input
                placeholder="https://…"
                value={form['destination_s3_endpoint'] ?? ''}
                onChange={e => set('destination_s3_endpoint', e.target.value)}
              />
            </Field>
            <div className="grid grid-cols-2 gap-4">
              <Field label="Region">
                <Input
                  placeholder="us-east-1"
                  value={form['destination_s3_region'] ?? ''}
                  onChange={e => set('destination_s3_region', e.target.value)}
                />
              </Field>
              <Field label="Bucket" required>
                <Input
                  placeholder="my-backups"
                  value={form['destination_s3_bucket'] ?? ''}
                  onChange={e => set('destination_s3_bucket', e.target.value)}
                />
              </Field>
            </div>
            <Field label="Prefix" hint="Optional folder prefix inside the bucket">
              <Input
                placeholder="supabase"
                value={form['destination_s3_prefix'] ?? ''}
                onChange={e => set('destination_s3_prefix', e.target.value)}
              />
            </Field>
            <Field label="Access Key ID" required>
              <Input
                value={form['destination_s3_access_key_id'] ?? ''}
                onChange={e => set('destination_s3_access_key_id', e.target.value)}
              />
            </Field>
            <Field label="Secret Access Key" required>
              <Input
                type="password"
                value={form['destination_s3_secret_access_key'] ?? ''}
                onChange={e => set('destination_s3_secret_access_key', e.target.value)}
              />
            </Field>
            <Toggle
              label="Force path style"
              description="Required for MinIO. Leave off for AWS S3 and R2."
              checked={form['destination_s3_force_path_style'] === 'true'}
              onChange={v => set('destination_s3_force_path_style', String(v))}
            />
          </>
        )}
      </Section>

      {/* Actions */}
      <div className="flex items-center gap-4 pb-4">
        <button
          onClick={() => save.mutate()}
          disabled={save.isPending}
          className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white text-sm font-medium rounded-lg transition-colors"
        >
          {save.isPending ? 'Saving…' : 'Save Settings'}
        </button>
        {saved && <span className="text-sm text-emerald-600 font-medium">✓ Saved successfully</span>}
        {save.isError && (
          <span className="text-sm text-red-600">{(save.error as Error).message}</span>
        )}
      </div>
    </div>
  )
}

// ── sub-components ────────────────────────────────────────────────────────────

function Section({ title, description, children }: {
  title: string
  description?: string
  children: React.ReactNode
}) {
  return (
    <div className="bg-white border border-gray-200 rounded-xl overflow-hidden">
      <div className="px-5 py-4 border-b border-gray-100">
        <h2 className="text-sm font-semibold text-gray-900">{title}</h2>
        {description && <p className="text-xs text-gray-500 mt-0.5">{description}</p>}
      </div>
      <div className="px-5 py-4 space-y-4">{children}</div>
    </div>
  )
}

function Field({ label, required, hint, children }: {
  label: string
  required?: boolean
  hint?: string
  children: React.ReactNode
}) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">
        {label}
        {required && <span className="text-red-500 ml-0.5">*</span>}
      </label>
      {children}
      {hint && <p className="mt-1 text-xs text-gray-400">{hint}</p>}
    </div>
  )
}

function Input({ className = '', ...props }: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      className={`w-full px-3 py-2 border border-gray-200 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent ${className}`}
    />
  )
}

function Toggle({ label, description, checked, onChange }: {
  label: string
  description: string
  checked: boolean
  onChange: (v: boolean) => void
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div>
        <p className="text-sm font-medium text-gray-700">{label}</p>
        <p className="text-xs text-gray-400 mt-0.5">{description}</p>
      </div>
      <button
        onClick={() => onChange(!checked)}
        className={`shrink-0 relative w-10 h-5 rounded-full transition-colors focus:outline-none mt-0.5 ${
          checked ? 'bg-indigo-600' : 'bg-gray-200'
        }`}
      >
        <span className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform ${
          checked ? 'translate-x-5' : 'translate-x-0'
        }`} />
      </button>
    </div>
  )
}
