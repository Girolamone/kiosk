import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode, TextareaHTMLAttributes } from 'react'

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: 'primary' | 'secondary' | 'ghost'
  busy?: boolean
}

export function Button({ variant = 'primary', busy, children, className = '', ...rest }: ButtonProps) {
  const base =
    'inline-flex items-center justify-center gap-2 rounded-md px-4 py-2.5 text-sm font-medium ' +
    'transition-all duration-150 disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0'
  const styles = {
    // Solid, not outlined. This is the one thing on the page that has to be
    // unmistakably clickable.
    primary: 'bg-accent text-white shadow-sm hover:bg-accent-hover hover:-translate-y-px active:translate-y-0',
    secondary: 'border border-line bg-raised text-ink hover:border-accent hover:bg-accent-soft',
    ghost: 'text-muted hover:text-ink',
  }[variant]

  return (
    <button className={`${base} ${styles} ${className}`} disabled={busy || rest.disabled} {...rest}>
      {busy && <Spinner />}
      {children}
    </button>
  )
}

export function Spinner() {
  return (
    <span
      className="inline-block size-3.5 animate-spin rounded-full border-2 border-current border-t-transparent"
      // Decorative: the surrounding text already says what is happening.
      aria-hidden="true"
    />
  )
}

export function Field({
  label,
  hint,
  children,
}: {
  label: string
  hint?: ReactNode
  children: ReactNode
}) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-sm font-medium text-ink">{label}</span>
      {children}
      {hint && <span className="mt-1.5 block text-xs text-muted">{hint}</span>}
    </label>
  )
}

const fieldStyles =
  'w-full rounded-md border border-line bg-raised px-3 py-2 text-sm text-ink transition-colors ' +
  'placeholder:text-muted/60 focus:border-accent'

export function Input({ className = '', ...rest }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={`${fieldStyles} ${className}`} {...rest} />
}

export function Textarea({ className = '', ...rest }: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={`${fieldStyles} ${className}`} {...rest} />
}

/**
 * role="alert" so a screen reader announces the problem when it appears.
 * A message that is only visible is a message half the users never get.
 */
export function ErrorNote({ children }: { children: ReactNode }) {
  if (!children) return null
  return (
    <p
      role="alert"
      className="rounded-md border border-red-600/25 bg-red-600/10 px-3 py-2 text-sm text-red-700 dark:text-red-300"
    >
      {children}
    </p>
  )
}

export function Card({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <div className={`rounded-lg border border-line bg-raised ${className}`}>{children}</div>
}

export function Empty({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="rounded-lg border border-dashed border-line px-6 py-12 text-center">
      <p className="font-display text-lg">{title}</p>
      {children && <div className="mt-2 text-sm text-muted">{children}</div>}
    </div>
  )
}
