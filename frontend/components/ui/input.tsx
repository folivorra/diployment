import * as React from 'react'
import { cn } from '@/lib/cn'

const baseInput = [
  'w-full rounded-lg border border-zinc-800 bg-zinc-950/50',
  'px-3 py-2 text-sm text-zinc-100 placeholder:text-zinc-600',
  'transition-[border-color,background,box-shadow] duration-150',
  'hover:border-zinc-700',
  'focus:outline-none focus:border-indigo-500/60 focus:bg-zinc-950',
  'focus:shadow-[0_0_0_3px_rgba(99,102,241,0.15)]',
  'disabled:opacity-50 disabled:cursor-not-allowed',
].join(' ')

export const Input = React.forwardRef<
  HTMLInputElement,
  React.InputHTMLAttributes<HTMLInputElement>
>(({ className, ...props }, ref) => (
  <input ref={ref} className={cn(baseInput, 'h-9', className)} {...props} />
))
Input.displayName = 'Input'

export const Textarea = React.forwardRef<
  HTMLTextAreaElement,
  React.TextareaHTMLAttributes<HTMLTextAreaElement>
>(({ className, ...props }, ref) => (
  <textarea
    ref={ref}
    className={cn(baseInput, 'resize-none font-mono text-xs leading-relaxed', className)}
    {...props}
  />
))
Textarea.displayName = 'Textarea'

export const Select = React.forwardRef<
  HTMLSelectElement,
  React.SelectHTMLAttributes<HTMLSelectElement>
>(({ className, children, ...props }, ref) => (
  <select
    ref={ref}
    className={cn(
      baseInput,
      'h-9 appearance-none bg-[length:14px] bg-[right_0.7rem_center] bg-no-repeat pr-9',
      "bg-[image:url(\"data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='14' height='14' viewBox='0 0 24 24' fill='none' stroke='%2371717a' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><polyline points='6 9 12 15 18 9'/></svg>\")]",
      className,
    )}
    {...props}
  >
    {children}
  </select>
))
Select.displayName = 'Select'

export function Label({ className, ...props }: React.LabelHTMLAttributes<HTMLLabelElement>) {
  return (
    <label
      className={cn('text-xs font-medium uppercase tracking-wider text-zinc-500', className)}
      {...props}
    />
  )
}

export function Field({
  label,
  htmlFor,
  hint,
  children,
  className,
}: {
  label?: string
  htmlFor?: string
  hint?: string
  children: React.ReactNode
  className?: string
}) {
  return (
    <div className={cn('flex flex-col gap-1.5', className)}>
      {label && <Label htmlFor={htmlFor}>{label}</Label>}
      {children}
      {hint && <p className="text-xs text-zinc-600">{hint}</p>}
    </div>
  )
}
