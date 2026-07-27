import { useSearchParams, useParams } from 'react-router'
import { useQuery } from 'urql'
import { StorefrontDocument, formatMoney, type ProductCardFragment } from '@kiosk/shared'
import { Button, Empty, Spinner } from '../components/ui'
import { CartPanel } from '../components/CartPanel'
import { useCart } from '../lib/cart'

export function Storefront() {
  const { slug = '' } = useParams()
  const [searchParams] = useSearchParams()
  const justPaid = searchParams.get('paid') === '1'

  const [{ data, fetching, error }] = useQuery({
    query: StorefrontDocument,
    variables: { slug },
  })
  const { cart, busy, setQuantity } = useCart(slug)

  if (fetching && !data) {
    return (
      <p className="flex items-center gap-2 py-16 text-muted">
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

  return (
    <div>
      <header className="border-b border-line pb-8">
        <h1 className="text-4xl">{store.name}</h1>
        {store.description && <p className="mt-2 max-w-xl text-muted">{store.description}</p>}
      </header>

      {justPaid && (
        <p className="mt-8 rounded-md bg-accent-soft px-4 py-3 text-sm">
          Thank you, your order is on its way. A receipt is in your inbox.
        </p>
      )}

      <div className="mt-8">
        {cart && cart.items.length > 0 && (
          <CartPanel cart={cart} busy={busy} onSetQuantity={setQuantity} />
        )}
      </div>

      {store.products.length === 0 ? (
        <div className="mt-8">
          <Empty title="Nothing for sale yet">
            The shop exists but has not published anything.
          </Empty>
        </div>
      ) : (
        <ul className="mt-8 grid gap-8 sm:grid-cols-2 lg:grid-cols-3">
          {store.products.map((product) => (
            <ProductTile
              key={product.id}
              product={product}
              currency={store.currency}
              inCart={cart?.items.find((item) => item.productId === product.id)?.quantity ?? 0}
              busy={busy}
              onAdd={(quantity) => setQuantity(product.id, quantity)}
            />
          ))}
        </ul>
      )}
    </div>
  )
}

function ProductTile({
  product,
  currency,
  inCart,
  busy,
  onAdd,
}: {
  product: ProductCardFragment
  currency: string
  inCart: number
  busy: boolean
  onAdd: (quantity: number) => void
}) {
  const image = product.images[0]

  return (
    <li>
      <div className="aspect-square overflow-hidden rounded-lg border border-line bg-white">
        {image ? (
          <img
            src={image.url}
            // The generated alt text lands here. An empty string would be
            // wrong: this image carries meaning, so it needs a description.
            alt={image.altText || product.name}
            className="size-full object-cover"
            loading="lazy"
          />
        ) : (
          <div className="flex size-full items-center justify-center text-sm text-muted">
            No photo
          </div>
        )}
      </div>

      <div className="mt-3 flex items-baseline justify-between gap-3">
        <h2 className="text-lg leading-snug">{product.name}</h2>
        <span className="shrink-0 text-sm tabular-nums text-muted">
          {formatMoney(product.priceCents, currency)}
        </span>
      </div>
      {product.description && (
        <p className="mt-1 line-clamp-3 text-sm text-muted">{product.description}</p>
      )}

      <Button
        variant="secondary"
        className="mt-3 w-full"
        disabled={busy}
        onClick={() => onAdd(inCart + 1)}
      >
        {inCart > 0 ? `In basket (${inCart})` : 'Add to basket'}
      </Button>
    </li>
  )
}
