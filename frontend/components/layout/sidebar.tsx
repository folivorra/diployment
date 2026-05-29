'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { motion } from 'motion/react'
import { LayoutDashboard, Plus, GitBranch, Copy, LogOut } from 'lucide-react'
import { Logo } from '@/components/layout/logo'
import { Button } from '@/components/ui/button'
import {
  Dropdown,
  DropdownContent,
  DropdownItem,
  DropdownLabel,
  DropdownSeparator,
  DropdownTrigger,
} from '@/components/ui/dropdown'
import { Tooltip, TooltipProvider } from '@/components/ui/tooltip'
import { toast } from 'sonner'
import { cn } from '@/lib/cn'
import { useProjects } from '@/lib/hooks/projects-context'
import type { User } from '@/lib/types'

interface SidebarProps {
  user: User
}

export function Sidebar({ user }: SidebarProps) {
  const pathname = usePathname()
  const projects = useProjects()
  const activeProjectId = pathname?.match(/^\/projects\/([^/]+)/)?.[1]

  return (
    <TooltipProvider delayDuration={300}>
      <aside className="sticky top-0 flex h-screen w-60 shrink-0 flex-col border-r border-zinc-800/80 bg-zinc-950/80 backdrop-blur">
        <div className="flex h-14 items-center px-5">
          <Link href="/dashboard" className="outline-none">
            <Logo />
          </Link>
        </div>

        <nav className="flex flex-col gap-0.5 px-3">
          <NavLink href="/dashboard" icon={<LayoutDashboard className="h-4 w-4" />} active={pathname === '/dashboard'}>
            Дашборд
          </NavLink>
        </nav>

        <div className="mt-6 flex items-center justify-between px-5">
          <span className="text-[10px] font-semibold uppercase tracking-wider text-zinc-500">Проекты</span>
          <Tooltip content="Импорт проекта" side="right">
            <Link
              href="/projects/new"
              className="flex h-5 w-5 items-center justify-center rounded text-zinc-500 transition-colors hover:bg-zinc-800 hover:text-zinc-200"
              aria-label="Импорт проекта"
            >
              <Plus className="h-3.5 w-3.5" />
            </Link>
          </Tooltip>
        </div>

        <div className="mt-2 flex-1 overflow-y-auto px-3">
          {projects.length === 0 ? (
            <p className="px-2 text-xs text-zinc-600">Проектов ещё нет.</p>
          ) : (
            <ul className="flex flex-col gap-0.5">
              {projects.map((p) => {
                const isActive = p.id === activeProjectId
                return (
                  <li key={p.id} className="relative">
                    <Link
                      href={`/projects/${p.id}/jobs`}
                      className={cn(
                        'group relative flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-colors',
                        isActive ? 'text-zinc-50' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200',
                      )}
                    >
                      {isActive && (
                        <motion.span
                          layoutId="sidebar-active"
                          className="absolute inset-0 rounded-md bg-zinc-800/70 ring-1 ring-zinc-700/50"
                          transition={{ type: 'spring', stiffness: 380, damping: 32 }}
                        />
                      )}
                      <GitBranch className="relative h-3.5 w-3.5 shrink-0 text-zinc-500 group-hover:text-zinc-300" />
                      <span className="relative truncate font-mono text-[12.5px]">
                        {p.repo_full_name.split('/').pop()}
                      </span>
                    </Link>
                  </li>
                )
              })}
            </ul>
          )}
        </div>

        <div className="border-t border-zinc-800/80 p-3">
          <Dropdown>
            <DropdownTrigger asChild>
              <button className="flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left transition-colors hover:bg-zinc-900">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img
                  src={user.avatar_url}
                  alt=""
                  className="h-7 w-7 rounded-full border border-zinc-800"
                />
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-xs font-medium text-zinc-200">
                    {user.login ?? `gh:${user.github_id}`}
                  </span>
                  <span className="truncate text-[10px] text-zinc-500">Авторизация через GitHub</span>
                </div>
              </button>
            </DropdownTrigger>
            <DropdownContent side="top" align="start">
              <DropdownLabel>Аккаунт</DropdownLabel>
              <DropdownItem
                onSelect={() => {
                  navigator.clipboard.writeText(user.id)
                  toast.success('ID пользователя скопирован')
                }}
              >
                <Copy className="h-3.5 w-3.5" />
                Скопировать ID
              </DropdownItem>
              <DropdownSeparator />
              <DropdownItem
                onSelect={() => {
                  // No logout endpoint on the backend yet; this just clears the cookie client-side
                  // by sending the user to the login page.
                  window.location.href = '/'
                }}
              >
                <LogOut className="h-3.5 w-3.5" />
                Выйти
              </DropdownItem>
            </DropdownContent>
          </Dropdown>
        </div>
      </aside>
    </TooltipProvider>
  )
}

function NavLink({
  href,
  icon,
  active,
  children,
}: {
  href: string
  icon: React.ReactNode
  active: boolean
  children: React.ReactNode
}) {
  return (
    <Link
      href={href}
      className={cn(
        'flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-colors',
        active ? 'bg-zinc-900 text-zinc-50' : 'text-zinc-400 hover:bg-zinc-900 hover:text-zinc-200',
      )}
    >
      <span className={active ? 'text-indigo-400' : 'text-zinc-500'}>{icon}</span>
      {children}
    </Link>
  )
}

export { Sidebar as default }

// Re-export Button to dodge unused-import lint if needed elsewhere
export { Button }
