'use client'

import * as React from 'react'
import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/cn'

const buttonVariants = cva(
  [
    'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-lg',
    'text-sm font-medium leading-none select-none',
    'transition-[background,color,border-color,box-shadow,transform] duration-150 ease-out',
    'disabled:pointer-events-none disabled:opacity-40',
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/60 focus-visible:ring-offset-2 focus-visible:ring-offset-zinc-950',
    'active:scale-[0.98]',
  ].join(' '),
  {
    variants: {
      variant: {
        primary: [
          'bg-indigo-500 text-white shadow-[0_1px_0_rgba(255,255,255,0.12)_inset,0_0_24px_-8px_rgba(99,102,241,0.6)]',
          'hover:bg-indigo-400 hover:shadow-[0_1px_0_rgba(255,255,255,0.16)_inset,0_0_32px_-6px_rgba(99,102,241,0.75)]',
        ].join(' '),
        secondary: [
          'border border-zinc-800 bg-zinc-900 text-zinc-100',
          'hover:bg-zinc-800 hover:border-zinc-700',
        ].join(' '),
        ghost: 'text-zinc-300 hover:bg-zinc-900 hover:text-zinc-50',
        danger: 'bg-red-500/90 text-white hover:bg-red-500',
        outline:
          'border border-zinc-800 bg-transparent text-zinc-200 hover:border-zinc-600 hover:text-zinc-50',
      },
      size: {
        sm: 'h-8 px-3 text-xs',
        md: 'h-9 px-4',
        lg: 'h-11 px-6 text-base',
        icon: 'h-9 w-9',
      },
    },
    defaultVariants: { variant: 'primary', size: 'md' },
  },
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button'
    return <Comp ref={ref} className={cn(buttonVariants({ variant, size }), className)} {...props} />
  },
)
Button.displayName = 'Button'

export { buttonVariants }
