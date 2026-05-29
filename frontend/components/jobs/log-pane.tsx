'use client'

import { useEffect, useRef, useState } from 'react'
import { AnimatePresence, motion } from 'motion/react'
import { CheckCircle2, Copy, Loader2, XCircle } from 'lucide-react'
import { toast } from 'sonner'
import { stripAnsi } from '@/lib/format'
import { cn } from '@/lib/cn'

interface LogPaneProps {
  title: string
  lines: string[]
  status: 'pending' | 'streaming' | 'success' | 'failed'
  emptyHint?: string
}

export function LogPane({ title, lines, status, emptyHint }: LogPaneProps) {
  const hint = emptyHint ?? defaultHint(status)
  const bottomRef = useRef<HTMLDivElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const [stickyBottom, setStickyBottom] = useState(true)

  useEffect(() => {
    if (stickyBottom) bottomRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' })
  }, [lines.length, stickyBottom])

  function onScroll() {
    const el = containerRef.current
    if (!el) return
    const distanceFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    setStickyBottom(distanceFromBottom < 80)
  }

  async function copyAll() {
    const text = lines.map(stripAnsi).join('\n')
    try {
      await navigator.clipboard.writeText(text)
      toast.success('Лог скопирован в буфер обмена')
    } catch {
      toast.error('Не удалось скопировать лог')
    }
  }

  const isStreaming = status === 'streaming'
  const isDone = status === 'success' || status === 'failed'

  return (
    <div className="overflow-hidden rounded-xl border border-zinc-800 bg-zinc-950">
      <div className="flex items-center justify-between border-b border-zinc-800 bg-zinc-900/50 px-4 py-2.5">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-zinc-200">{title}</span>
          {isStreaming && (
            <span className="flex items-center gap-1 text-[11px] text-indigo-300">
              <Loader2 className="h-3 w-3 animate-spin" />
              Стрим
            </span>
          )}
          {isDone && status === 'success' && (
            <span className="flex items-center gap-1 text-[11px] text-emerald-400">
              <CheckCircle2 className="h-3 w-3" />
              Готово
            </span>
          )}
          {isDone && status === 'failed' && (
            <span className="flex items-center gap-1 text-[11px] text-red-400">
              <XCircle className="h-3 w-3" />
              Упало
            </span>
          )}
        </div>
        <button
          onClick={copyAll}
          disabled={lines.length === 0}
          className="flex items-center gap-1.5 rounded-md border border-zinc-800 px-2 py-1 text-[11px] text-zinc-400 transition-colors hover:border-zinc-700 hover:text-zinc-200 disabled:opacity-30 disabled:hover:border-zinc-800 disabled:hover:text-zinc-400"
        >
          <Copy className="h-3 w-3" />
          Скопировать
        </button>
      </div>

      <div
        ref={containerRef}
        onScroll={onScroll}
        className="h-80 overflow-y-auto bg-zinc-950 px-4 py-3 font-mono text-[12px] leading-relaxed"
      >
        {lines.length === 0 ? (
          <span className="text-zinc-600">{hint}</span>
        ) : (
          <AnimatePresence initial={false}>
            {lines.map((line, i) => (
              <motion.div
                key={i}
                initial={{ opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.12 }}
                className={cn('whitespace-pre-wrap break-all text-zinc-300')}
              >
                {stripAnsi(line)}
              </motion.div>
            ))}
          </AnimatePresence>
        )}
        <div ref={bottomRef} />
      </div>
    </div>
  )
}

function defaultHint(status: LogPaneProps['status']): string {
  switch (status) {
    case 'streaming':
      return 'Ждём логи…'
    case 'failed':
      return 'Упало до первой строки вывода.'
    case 'success':
      return 'Без вывода.'
    case 'pending':
      return 'Ещё не запущено.'
  }
}
