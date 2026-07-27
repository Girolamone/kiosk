import { useState } from 'react'
import { useNavigate } from 'react-router'
import { Button, ErrorNote, Field, Input } from '../components/ui'
import { readableError, useLogIn, useSession, useSignUp } from '../lib/session'

export function SignIn() {
  const [mode, setMode] = useState<'signin' | 'signup'>('signin')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  const [logInState, logIn] = useLogIn()
  const [signUpState, signUp] = useSignUp()
  const { reload } = useSession()
  const navigate = useNavigate()

  const busy = logInState.fetching || signUpState.fetching
  const error = readableError(logInState.error ?? signUpState.error)

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    const run = mode === 'signin' ? logIn : signUp
    const result = await run({ email, password })
    if (result.error) return

    // The session is now a cookie the browser holds. Re-ask the server who we
    // are so the header updates, then move on.
    reload()
    navigate('/dashboard')
  }

  return (
    <div className="mx-auto max-w-sm py-8">
      <h1 className="text-3xl">{mode === 'signin' ? 'Sign in' : 'Create an account'}</h1>
      <p className="mt-2 text-sm text-muted">
        {mode === 'signin'
          ? 'To manage your shop and its products.'
          : 'A shop takes about a minute to set up.'}
      </p>

      <form onSubmit={submit} className="mt-6 space-y-4">
        <Field label="Email">
          <Input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoComplete="email"
            required
          />
        </Field>

        <Field label="Password" hint={mode === 'signup' ? 'At least 8 characters.' : undefined}>
          <Input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete={mode === 'signin' ? 'current-password' : 'new-password'}
            required
            minLength={mode === 'signup' ? 8 : undefined}
          />
        </Field>

        <ErrorNote>{error}</ErrorNote>

        <Button type="submit" busy={busy} className="w-full">
          {mode === 'signin' ? 'Sign in' : 'Create account'}
        </Button>
      </form>

      <p className="mt-6 text-center text-sm text-muted">
        {mode === 'signin' ? 'No account yet?' : 'Already have one?'}{' '}
        <button
          type="button"
          onClick={() => setMode(mode === 'signin' ? 'signup' : 'signin')}
          className="text-accent underline"
        >
          {mode === 'signin' ? 'Create one' : 'Sign in'}
        </button>
      </p>
    </div>
  )
}
