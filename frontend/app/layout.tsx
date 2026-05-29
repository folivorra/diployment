import type { Metadata } from 'next'
import { Toaster } from 'sonner'
import './globals.css'

export const metadata: Metadata = {
  title: 'diployment',
  description: 'A focused CI/CD platform for GitHub projects',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-zinc-950 font-sans text-zinc-100 antialiased">
        {children}
        <Toaster
          theme="dark"
          position="bottom-right"
          toastOptions={{
            classNames: {
              toast:
                'group rounded-lg border border-zinc-800 bg-zinc-900/95 backdrop-blur shadow-[0_8px_32px_-8px_rgba(0,0,0,0.5)]',
              title: 'text-sm text-zinc-100',
              description: 'text-xs text-zinc-500',
            },
          }}
        />
      </body>
    </html>
  )
}
