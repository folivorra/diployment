'use client'

import { motion } from 'motion/react'
import { ArrowRight, GitBranch, Rocket, Activity } from 'lucide-react'
import { Logo } from '@/components/layout/logo'
import { getApiUrl } from '@/lib/api'

export default function Landing() {
  return (
    <main className="relative flex min-h-screen flex-col overflow-hidden">
      <div className="pointer-events-none absolute inset-0 bg-radial-violet" />
      <div className="pointer-events-none absolute inset-0 bg-grid opacity-50 [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_20%,transparent_70%)]" />

      {/* glow blob */}
      <motion.div
        aria-hidden
        initial={{ opacity: 0, scale: 0.8 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 1.4, ease: [0.22, 1, 0.36, 1] }}
        className="pointer-events-none absolute left-1/2 top-[-15%] h-[440px] w-[820px] -translate-x-1/2 rounded-full bg-gradient-to-b from-indigo-600/30 via-violet-600/15 to-transparent blur-3xl"
      />

      <header className="relative z-10 flex items-center justify-between px-8 py-6">
        <Logo />
        <a
          href={`${getApiUrl()}/auth/login`}
          className="text-xs text-zinc-500 transition-colors hover:text-zinc-300"
        >
          Войти
        </a>
      </header>

      <section className="relative z-10 flex flex-1 flex-col items-center justify-center px-8 pb-24 text-center">
        <motion.span
          initial={{ opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
          className="mb-6 inline-flex items-center gap-2 rounded-full border border-zinc-800 bg-zinc-900/60 px-3 py-1 text-[11px] text-zinc-400 backdrop-blur"
        >
          <span className="relative inline-flex h-1.5 w-1.5">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 animate-pulse-dot" />
          </span>
          CI/CD на базе GitHub — без YAML-конфигов
        </motion.span>

        <motion.h1
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.05, ease: [0.22, 1, 0.36, 1] }}
          className="text-balance bg-gradient-to-b from-zinc-50 to-zinc-400 bg-clip-text text-5xl font-semibold tracking-tight text-transparent md:text-6xl"
        >
          Деплой на каждый коммит.
          <br />
          <span className="bg-gradient-to-r from-indigo-300 to-violet-400 bg-clip-text text-transparent">
            В реальном времени.
          </span>
        </motion.h1>

        <motion.p
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.12, ease: [0.22, 1, 0.36, 1] }}
          className="mt-5 max-w-xl text-pretty text-base text-zinc-400"
        >
          Подключаешь GitHub-репозиторий, указываешь SSH-цель — diployment собирает Docker-образ
          и катит его на сервер при каждом push. Живые логи, без склеек руками.
        </motion.p>

        <motion.div
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.18, ease: [0.22, 1, 0.36, 1] }}
          className="mt-9"
        >
          <a
            href={`${getApiUrl()}/auth/login`}
            className="group inline-flex h-11 items-center gap-2 rounded-lg bg-zinc-50 px-6 text-sm font-semibold text-zinc-950 transition-all duration-150 hover:bg-white hover:shadow-[0_0_32px_-4px_rgba(255,255,255,0.4)] active:scale-[0.98]"
          >
            <svg className="h-4 w-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z" />
            </svg>
            Войти через GitHub
            <ArrowRight className="h-4 w-4 transition-transform duration-150 group-hover:translate-x-0.5" />
          </a>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.7, delay: 0.32, ease: [0.22, 1, 0.36, 1] }}
          className="mt-24 grid w-full max-w-3xl grid-cols-1 gap-3 md:grid-cols-3"
        >
          <Feature
            icon={<GitBranch className="h-4 w-4" />}
            title="Push — и поехали"
            desc="Webhook на каждый коммит сам запускает сборку."
          />
          <Feature
            icon={<Rocket className="h-4 w-4" />}
            title="SSH-выкатка"
            desc="Артефакт заливается на твой хост, сервис перезапускается."
          />
          <Feature
            icon={<Activity className="h-4 w-4" />}
            title="Логи в реальном времени"
            desc="Стрим вывода сборки и деплоя — построчно, по мере выполнения."
          />
        </motion.div>
      </section>
    </main>
  )
}

function Feature({ icon, title, desc }: { icon: React.ReactNode; title: string; desc: string }) {
  return (
    <div className="rounded-xl border border-zinc-800/80 bg-zinc-900/30 p-4 text-left backdrop-blur-sm">
      <div className="mb-2 inline-flex h-7 w-7 items-center justify-center rounded-md border border-zinc-800 bg-zinc-900 text-indigo-300">
        {icon}
      </div>
      <p className="text-sm font-medium text-zinc-100">{title}</p>
      <p className="mt-1 text-xs text-zinc-500">{desc}</p>
    </div>
  )
}
