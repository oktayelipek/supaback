import { NavLink } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { cx } from '../lib/utils'

export function Navbar() {
  const { data } = useQuery({
    queryKey: ['health'],
    queryFn: () => fetch('/api/health').then(r => r.json()) as Promise<{ configured: boolean }>,
    refetchInterval: 30_000,
  })

  return (
    <header className="bg-white border-b border-gray-200 sticky top-0 z-40">
      <div className="max-w-6xl mx-auto px-4 sm:px-6 flex items-center h-14 gap-8">
        <div className="flex items-center gap-2">
          <div className="w-7 h-7 rounded-md bg-indigo-600 flex items-center justify-center">
            <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 5.625c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
            </svg>
          </div>
          <span className="font-semibold text-gray-900 text-sm">SupaBack</span>
        </div>

        <nav className="flex items-center gap-1 flex-1">
          {[
            { to: '/', label: 'Dashboard' },
            { to: '/schedules', label: 'Schedules' },
          ].map(({ to, label }) => (
            <NavLink
              key={to}
              to={to}
              end
              className={({ isActive }) => cx(
                'px-3 py-1.5 rounded-md text-sm font-medium transition-colors',
                isActive
                  ? 'bg-gray-100 text-gray-900'
                  : 'text-gray-500 hover:text-gray-900 hover:bg-gray-50'
              )}
            >
              {label}
            </NavLink>
          ))}
        </nav>

        <NavLink
          to="/settings"
          className={({ isActive }) => cx(
            'flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-colors',
            isActive
              ? 'bg-gray-100 text-gray-900'
              : 'text-gray-500 hover:text-gray-900 hover:bg-gray-50'
          )}
        >
          {data && !data.configured && (
            <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse" />
          )}
          Settings
        </NavLink>
      </div>

      {/* Setup banner */}
      {data && !data.configured && (
        <div className="bg-amber-50 border-b border-amber-200 px-4 py-2 text-center text-xs text-amber-800">
          SupaBack is not configured.{' '}
          <NavLink to="/settings" className="font-semibold underline underline-offset-2">
            Enter your Supabase credentials
          </NavLink>{' '}
          to enable backups.
        </div>
      )}
    </header>
  )
}
