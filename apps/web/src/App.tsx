import { Link, Route, Routes, useMatch, useNavigate } from 'react-router'
import { useLogOut, useSession } from './lib/session'
import { CartProvider } from './lib/cart'
import { useCart } from './lib/cart-context'
import { useTheme } from './lib/theme'
import { Button } from './components/ui'
import { Home } from './routes/Home'
import { Storefront } from './routes/Storefront'
import { ProductDetail } from './routes/ProductDetail'
import { SignIn } from './routes/SignIn'
import { Dashboard } from './routes/Dashboard'
import { StoreEditor } from './routes/StoreEditor'

export function App() {
  return (
    <CartProvider>
      <div className="min-h-dvh">
        <Header />
        <main className="mx-auto w-full max-w-7xl px-6 py-10">
          <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/s/:slug" element={<Storefront />} />
          <Route path="/s/:slug/p/:id" element={<ProductDetail />} />
          <Route path="/signin" element={<SignIn />} />
          <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/dashboard/:slug" element={<StoreEditor />} />
            <Route path="*" element={<NotFound />} />
          </Routes>
        </main>
      </div>
    </CartProvider>
  )
}

function Header() {
  const { user, reload } = useSession()
  const [, logOut] = useLogOut()
  const navigate = useNavigate()

  const inStore = useMatch({ path: '/s/:slug', end: false })
  const slug = inStore?.params.slug ?? ''
  const { itemCount } = useCart()

  async function signOut() {
    await logOut({})
    reload()
    navigate('/')
  }

  return (
    <header className="sticky top-0 z-10 border-b border-line bg-paper/85 backdrop-blur">
      <div className="mx-auto flex w-full max-w-7xl items-center justify-between px-6 py-4">
        <Link to="/" className="font-display text-xl transition-opacity hover:opacity-70">
          Kiosk
        </Link>

        <nav className="flex items-center gap-4 text-sm">
          <ThemeToggle />

          {slug && itemCount > 0 && (
            <Link
              to={`/s/${slug}`}
              className="flex items-center gap-2 rounded-full bg-accent-soft px-3 py-1.5 text-accent transition-colors hover:bg-accent hover:text-white"
            >
              <BasketIcon />
              <span className="tabular-nums">{itemCount}</span>
              <span className="sr-only">items in your basket</span>
            </Link>
          )}

          {user ? (
            <>
              <Link to="/dashboard" className="text-muted transition-colors hover:text-ink">
                {user.email}
              </Link>
              <Button variant="ghost" onClick={signOut}>
                Sign out
              </Button>
            </>
          ) : (
            <Link to="/signin" className="text-muted transition-colors hover:text-ink">
              Sign in
            </Link>
          )}
        </nav>
      </div>
    </header>
  )
}

function ThemeToggle() {
  const { theme, toggle } = useTheme()
  const goingDark = theme === 'light'

  return (
    <button
      type="button"
      onClick={toggle}
      // The label says what pressing it does, not what the state is. A
      // screen reader reading "light" cannot tell which of the two that is.
      aria-label={goingDark ? 'Switch to dark' : 'Switch to light'}
      title={goingDark ? 'Switch to dark' : 'Switch to light'}
      className="rounded-full p-2 text-muted transition-colors hover:bg-accent-soft hover:text-ink"
    >
      {/* Nudged down a pixel. The icon is centred exactly, but lowercase
          text reads lower than its box, so a geometrically centred icon
          looks high beside it. */}
      <span className="block translate-y-px">{goingDark ? <MoonIcon /> : <SunIcon />}</span>
    </button>
  )
}

function MoonIcon() {
  return (
    <svg viewBox="0 0 24 24" className="size-4" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true">
      <path d="M20 14.5A8.5 8.5 0 0 1 9.5 4a8.5 8.5 0 1 0 10.5 10.5Z" strokeLinejoin="round" />
    </svg>
  )
}

function SunIcon() {
  return (
    <svg viewBox="0 0 24 24" className="size-4" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true">
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M2 12h2M20 12h2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M19.1 4.9l-1.4 1.4M6.3 17.7l-1.4 1.4" strokeLinecap="round" />
    </svg>
  )
}

function BasketIcon() {
  return (
    <svg viewBox="0 0 24 24" className="size-4" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
      <path d="M4 8h16l-1.2 11a2 2 0 0 1-2 1.8H7.2a2 2 0 0 1-2-1.8L4 8Z" strokeLinejoin="round" />
      <path d="M9 8V6a3 3 0 0 1 6 0v2" strokeLinecap="round" />
    </svg>
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
