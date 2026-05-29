const ANSI_RE = /\x1b\[[0-9;]*[mGKHF]/g

export function stripAnsi(s: string): string {
  return s.replace(ANSI_RE, '').replace(/\r/g, '').replace(/\n$/, '')
}

export function shortSha(sha: string): string {
  return sha.slice(0, 7)
}

const RTF = new Intl.RelativeTimeFormat('ru', { numeric: 'auto' })

export function timeAgo(input: string | Date): string {
  const date = typeof input === 'string' ? new Date(input) : input
  const diff = (date.getTime() - Date.now()) / 1000

  const units: [Intl.RelativeTimeFormatUnit, number][] = [
    ['year',   60 * 60 * 24 * 365],
    ['month',  60 * 60 * 24 * 30],
    ['day',    60 * 60 * 24],
    ['hour',   60 * 60],
    ['minute', 60],
    ['second', 1],
  ]

  for (const [unit, secs] of units) {
    if (Math.abs(diff) >= secs || unit === 'second') {
      return RTF.format(Math.round(diff / secs), unit)
    }
  }
  return ''
}

export function formatDuration(startISO?: string, endISO?: string): string | null {
  if (!startISO || !endISO) return null
  const ms = new Date(endISO).getTime() - new Date(startISO).getTime()
  if (ms < 0 || !Number.isFinite(ms)) return null

  const totalSec = Math.round(ms / 1000)
  if (totalSec < 60) return `${totalSec}с`
  const mins = Math.floor(totalSec / 60)
  const secs = totalSec % 60
  if (mins < 60) return secs ? `${mins}м ${secs}с` : `${mins}м`
  const hrs = Math.floor(mins / 60)
  const rmins = mins % 60
  return rmins ? `${hrs}ч ${rmins}м` : `${hrs}ч`
}

export function formatDateTime(input: string | Date): string {
  const date = typeof input === 'string' ? new Date(input) : input
  return date.toLocaleString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

export function githubCommitUrl(cloneUrl: string, sha: string): string | null {
  // https://github.com/owner/repo.git → https://github.com/owner/repo/commit/sha
  try {
    const url = new URL(cloneUrl.replace(/\.git$/, ''))
    return `${url.origin}${url.pathname}/commit/${sha}`
  } catch {
    return null
  }
}
