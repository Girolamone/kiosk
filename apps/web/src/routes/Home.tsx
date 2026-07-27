import { Link } from 'react-router'

export function Home() {
  return (
    <div className="mx-auto max-w-2xl py-10">
      <h1 className="text-4xl leading-tight">
        A storefront that writes its own product pages.
      </h1>
      <p className="mt-4 text-muted">
        Photograph the thing you are selling. Kiosk reads the photo and drafts
        the title, the description and the alt text, then you edit what it got
        wrong and publish.
      </p>

      <div className="mt-8 flex flex-wrap gap-3">
        <Link
          to="/s/pine-and-salt"
          className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent/90"
        >
          See a live storefront
        </Link>
        <Link
          to="/dashboard"
          className="rounded-md border border-line bg-white px-4 py-2 text-sm font-medium hover:bg-accent-soft"
        >
          Open the seller dashboard
        </Link>
      </div>

      <dl className="mt-14 grid gap-8 border-t border-line pt-8 sm:grid-cols-3">
        <div>
          <dt className="font-display text-lg">Go and GraphQL</dt>
          <dd className="mt-1 text-sm text-muted">
            One schema, generated into typed clients for web and mobile.
          </dd>
        </div>
        <div>
          <dt className="font-display text-lg">Postgres</dt>
          <dd className="mt-1 text-sm text-muted">
            Prices as integer cents, orders that keep what was actually bought.
          </dd>
        </div>
        <div>
          <dt className="font-display text-lg">Gemini</dt>
          <dd className="mt-1 text-sm text-muted">
            Listing copy from a photograph, and a form that still works without
            it.
          </dd>
        </div>
      </dl>
    </div>
  )
}
