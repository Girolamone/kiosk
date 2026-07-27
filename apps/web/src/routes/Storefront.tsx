import { Link, useParams, useSearchParams } from 'react-router'
import { useQuery } from 'urql'
import { StorefrontDocument, formatMoney, type ProductCardFragment } from '@kiosk/shared'
import { Button, Empty, Spinner } from '../components/ui'
import { CartPanel } from '../components/CartPanel'
import { useCart } from '../lib/cart-context'

export function Storefront() {
  const { slug = '' } = useParams()
  const [searchParams] = useSearchParams()
  const justPaid = searchParams.get('paid') === '1'

  const [{ data, fetching, error }] = useQuery({
    query: StorefrontDocument,
    variables: { slug },
  })
  const { cart, busy, setQuantity } = useCart()

  if (fetching && !data) {
    return (
      <p className="flex items-center gap-2 py-24 text-muted">
        <Spinner /> Loading the shop…
      </p>
    )
  }
  if (error) {
    return <Empty title="That shop could not be loaded">{error.message}</Empty>
  }

  const store = data?.store
  if (!store) {
    return (
      <Empty title="No shop at this address">
        Nobody has claimed <span className="font-mono">{slug}</span> yet.
      </Empty>
    )
  }

  const lead = store.products[0]

  return (
    <div>
      {/* The lead print is the masthead. A shop selling photographs that
          opens on anything other than a photograph wastes its first screen. */}
      {lead?.images[0] ? (
        <section className="relative -mx-6 mb-20 h-[68vh] min-h-96 overflow-hidden">
          <img
            src={lead.images[0].url}
            alt={lead.images[0].altText || lead.name}
            className="size-full object-cover"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-dark/85 via-dark/30 to-transparent" />

          <div className="absolute inset-x-0 bottom-0 p-8 sm:p-12">
            <p className="label text-white/70">Print shop</p>
            <h1 className="mt-3 max-w-4xl font-display text-6xl leading-[0.95] text-white sm:text-8xl">
              {store.name}
            </h1>
            {store.description && (
              <p className="mt-5 max-w-md text-white/80">{store.description}</p>
            )}
          </div>
        </section>
      ) : (
        <header className="mb-16 border-b border-line pb-10">
          <h1 className="font-display text-6xl">{store.name}</h1>
          {store.description && <p className="mt-3 text-lg text-muted">{store.description}</p>}
        </header>
      )}

      {justPaid && (
        <p className="mb-12 border-l-2 border-accent bg-accent-soft px-5 py-4 text-accent">
          Thank you, your order is on its way. A receipt is in your inbox.
        </p>
      )}

      {cart && cart.items.length > 0 && (
        <div className="mb-20">
          <CartPanel cart={cart} busy={busy} onSetQuantity={setQuantity} />
        </div>
      )}

      {store.products.length === 0 ? (
        <Empty title="Nothing for sale yet">
          The shop exists but has not published anything.
        </Empty>
      ) : (
        <>
          <div className="mb-10 flex items-baseline justify-between border-b border-ink pb-4">
            <h2 className="label">The collection</h2>
            <span className="label text-muted">
              {store.products.length} print{store.products.length === 1 ? '' : 's'}
            </span>
          </div>

          {/* Deliberately uneven. Every print at the same size reads as a
              catalogue page; letting some run wide reads as a curation. */}
          <ul className="grid gap-x-8 gap-y-20 sm:grid-cols-6">
            {store.products.map((product, index) => (
              <ProductTile
                key={product.id}
                slug={slug}
                index={index}
                product={product}
                currency={store.currency}
                wide={index % 5 === 0 || index % 5 === 3}
                inCart={cart?.items.find((item) => item.productId === product.id)?.quantity ?? 0}
                busy={busy}
                onAdd={(quantity) => setQuantity(product.id, quantity)}
              />
            ))}
          </ul>
        </>
      )}
    </div>
  )
}

function ProductTile({
  slug,
  index,
  product,
  currency,
  wide,
  inCart,
  busy,
  onAdd,
}: {
  slug: string
  index: number
  product: ProductCardFragment
  currency: string
  wide: boolean
  inCart: number
  busy: boolean
  onAdd: (quantity: number) => void
}) {
  const image = product.images[0]
  const plate = String(index + 1).padStart(2, '0')

  return (
    <li className={`group ${wide ? 'sm:col-span-4' : 'sm:col-span-2'}`}>
      <Link to={`/s/${slug}/p/${product.id}`} className="block">
        {/* One height for every print, whatever its column span. Mixed
            aspect ratios pushed the captions of neighbouring tiles to
            different heights, which reads as a mistake rather than a
            rhythm. A wide print is a wider crop, not a taller one. */}
        <div className="h-[26rem] overflow-hidden bg-raised">
          {image ? (
            <img
              src={image.url}
              // The generated alt text lands here. An empty string would be
              // wrong: this image carries meaning, so it needs a description.
              alt={image.altText || product.name}
              className="size-full object-cover transition-transform duration-700 ease-out group-hover:scale-[1.03]"
              loading={index < 3 ? 'eager' : 'lazy'}
            />
          ) : (
            <div className="flex size-full items-center justify-center text-sm text-muted">
              No photo
            </div>
          )}
        </div>

        <div className="mt-5 flex items-start justify-between gap-6 border-t border-ink pt-3">
          <div className="min-w-0">
            <p className="label text-muted">Plate {plate}</p>
            <h3 className="mt-1.5 font-display text-2xl leading-tight transition-colors group-hover:text-accent">
              {product.name}
            </h3>
          </div>
          <span className="shrink-0 font-mono text-sm tabular-nums">
            {formatMoney(product.priceCents, currency)}
          </span>
        </div>

        {product.description && (
          <p className="mt-2 line-clamp-2 text-sm leading-relaxed text-muted">
            {product.description}
          </p>
        )}
      </Link>

      <Button
        variant={inCart > 0 ? 'secondary' : 'primary'}
        className="mt-4 w-full"
        disabled={busy}
        onClick={() => onAdd(inCart + 1)}
      >
        {inCart > 0 ? `In basket (${inCart}) — add another` : 'Add to basket'}
      </Button>
    </li>
  )
}
