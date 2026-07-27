import { useState } from 'react'
import { Link, Navigate } from 'react-router'
import { useMutation, useQuery } from 'urql'
import { CreateStoreDocument, MyStoresDocument } from '@kiosk/shared'
import { Button, Card, Empty, ErrorNote, Field, Input, Spinner } from '../components/ui'
import { readableError, useSession } from '../lib/session'

export function Dashboard() {
  const { user, loading } = useSession()
  const [{ data, fetching }, refetchStores] = useQuery({ query: MyStoresDocument })

  if (loading) {
    return (
      <p className="flex items-center gap-2 py-16 text-muted">
        <Spinner /> Checking your session…
      </p>
    )
  }
  if (!user) return <Navigate to="/signin" replace />

  const stores = data?.myStores ?? []

  return (
    <div>
      <h1 className="text-3xl">Your shops</h1>

      {fetching && !data ? (
        <p className="mt-8 flex items-center gap-2 text-muted">
          <Spinner /> Loading…
        </p>
      ) : stores.length === 0 ? (
        <div className="mt-8">
          <Empty title="No shop yet">Create one below and it goes live immediately.</Empty>
        </div>
      ) : (
        <ul className="mt-8 grid gap-4 sm:grid-cols-2">
          {stores.map((store) => (
            <li key={store.id}>
              <Card className="p-5">
                <h2 className="text-xl">{store.name}</h2>
                <p className="mt-1 font-mono text-xs text-muted">/s/{store.slug}</p>
                {store.description && <p className="mt-2 text-sm text-muted">{store.description}</p>}
                <div className="mt-4 flex gap-3 text-sm">
                  <Link to={`/dashboard/${store.slug}`} className="text-accent underline">
                    Manage products
                  </Link>
                  <Link to={`/s/${store.slug}`} className="text-muted underline">
                    View shop
                  </Link>
                </div>
              </Card>
            </li>
          ))}
        </ul>
      )}

      <NewStoreForm onCreated={() => refetchStores({ requestPolicy: 'network-only' })} />
    </div>
  )
}

function NewStoreForm({ onCreated }: { onCreated: () => void }) {
  const [name, setName] = useState('')
  const [slug, setSlug] = useState('')
  const [slugEdited, setSlugEdited] = useState(false)
  const [{ fetching, error }, createStore] = useMutation(CreateStoreDocument)

  // The slug follows the name until the seller touches it, then it is theirs.
  // Rewriting it after that would silently undo their edit.
  function changeName(value: string) {
    setName(value)
    if (!slugEdited) setSlug(slugify(value))
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    const result = await createStore({ input: { name, slug } })
    if (result.error) return
    setName('')
    setSlug('')
    setSlugEdited(false)
    onCreated()
  }

  return (
    <section className="mt-14 border-t border-line pt-8">
      <h2 className="text-2xl">Open a shop</h2>
      <form onSubmit={submit} className="mt-4 max-w-md space-y-4">
        <Field label="Name">
          <Input value={name} onChange={(e) => changeName(e.target.value)} required />
        </Field>

        <Field label="Address" hint={<>Your shop will live at /s/{slug || 'your-shop'}</>}>
          <Input
            value={slug}
            onChange={(e) => {
              setSlugEdited(true)
              setSlug(e.target.value)
            }}
            pattern="[a-z0-9][a-z0-9-]{1,38}[a-z0-9]"
            required
          />
        </Field>

        <ErrorNote>{readableError(error)}</ErrorNote>

        <Button type="submit" busy={fetching}>
          Create shop
        </Button>
      </form>
    </section>
  )
}

function slugify(value: string): string {
  return value
    .toLowerCase()
    .normalize('NFD')
    // Strip the combining marks NFD just separated out, so "Café Nord"
    // becomes "cafe-nord" rather than losing the letter entirely.
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 40)
}
