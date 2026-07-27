import { Link, Route, Routes, useNavigate } from 'react-router'
import { useLogOut, useSession } from './lib/session'
import { Button } from './components/ui'
import { Home } from './routes/Home'
import { Storefront } from './routes/Storefront'
import { SignIn } from './routes/SignIn'
import { Dashboard } from './routes/Dashboard'
import { StoreEditor } from './routes/StoreEditor'

export function App() {
  return (
    <div className="min-h-dvh">
      <Header />
      <main className="mx-auto w-full max-w-5xl px-5 py-10">
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/s/:slug" element={<Storefront />} />
          <Route path="/signin" element={<SignIn />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/dashboard/:slug" element={<StoreEditor />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </main>
    </div>
  )
}

function Header() {
  const { user, reload } = useSession()
  const [, logOut] = useLogOut()
  const navigate = useNavigate()

  async function signOut() {
    await logOut({})
    reload()
    navigate('/')
  }

  return (
    <header className="border-b border-line">
      <div className="mx-auto flex w-full max-w-5xl items-center justify-between px-5 py-4">
        <Link to="/" className="font-display text-xl">
          Kiosk
        </Link>
        <nav className="flex items-center gap-3 text-sm">
          {user ? (
            <>
              <Link to="/dashboard" className="text-muted hover:text-ink">
                {user.email}
              </Link>
              <Button variant="ghost" onClick={signOut}>
                Sign out
              </Button>
            </>
          ) : (
            <Link to="/signin" className="text-muted hover:text-ink">
              Sign in
            </Link>
          )}
        </nav>
      </div>
    </header>
  )
}

function NotFound() {
  return (
    <div className="py-16 text-center">
      <h1 className="text-3xl">Nothing here</h1>
      <p className="mt-2 text-muted">That page does not exist.</p>
      <Link to="/" className="mt-6 inline-block text-accent underline">
        Back to the start
      </Link>
    </div>
  )
}
