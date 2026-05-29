'use client'

import Link from 'next/link'
import { ChevronRight } from 'lucide-react'
import { cn } from '@/lib/cn'

export interface Crumb {
  label: string
  href?: string
  mono?: boolean
}

export function Breadcrumbs({ items }: { items: Crumb[] }) {
  return (
    <nav className="flex items-center gap-1.5 text-xs text-zinc-500">
      {items.map((c, i) => {
        const isLast = i === items.length - 1
        const content = (
          <span className={cn(c.mono && 'font-mono', isLast ? 'text-zinc-200' : '')}>{c.label}</span>
        )
        return (
          <span key={i} className="flex items-center gap-1.5">
            {c.href && !isLast ? (
              <Link href={c.href} className="transition-colors hover:text-zinc-300">
                {content}
              </Link>
            ) : (
              content
            )}
            {!isLast && <ChevronRight className="h-3 w-3 text-zinc-700" />}
          </span>
        )
      })}
    </nav>
  )
}
