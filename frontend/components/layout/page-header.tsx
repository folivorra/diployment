'use client'

import { motion } from 'motion/react'
import { cn } from '@/lib/cn'

interface PageHeaderProps {
  title: React.ReactNode
  description?: React.ReactNode
  actions?: React.ReactNode
  meta?: React.ReactNode
  className?: string
}

export function PageHeader({ title, description, actions, meta, className }: PageHeaderProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: -6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
      className={cn('mb-8 flex flex-wrap items-end justify-between gap-4', className)}
    >
      <div className="flex flex-col gap-1">
        <h1 className="flex items-center gap-3 text-2xl font-semibold tracking-tight text-zinc-50">
          {title}
        </h1>
        {description && <p className="max-w-2xl text-sm text-zinc-500">{description}</p>}
        {meta && <div className="mt-1 flex items-center gap-2">{meta}</div>}
      </div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </motion.div>
  )
}
