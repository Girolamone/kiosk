import { Link, useParams } from 'react-router'
import { useQuery } from 'urql'
import { ProductDetailDocument, StorefrontDocument, formatMoney } from '@kiosk/shared'
import { Button, Empty, Spinner } from '../components/ui'
import { useCart } from '../lib/cart-context'

export function ProductDetail() {
  const { slug = '', id = '' } = useParams()

  const [{ data, fetching, error }] = useQuery({
    query: ProductDetailDocument,
    variables: { id },
  })
  // The shop supplies the currency and the name for the trail back. It is
  // almost always already cached from the storefront the visitor came from.
  const [{ data: storeData }] = useQuery({
    query: StorefrontDocument,
    variables: { slug },
  })

  const { cart, busy, setQuantity } = useCart()

  if (fetching && !data) {
    return (
      <p className="flex items-center gap-2 py-16 text-muted">
        <Spinner /> Loading…
      </p>
    )
  }
  if (error) {
    return <Empty title="That print could not be loaded">{error.message}</Empty>
  }

  const product = data?.product
  if (!product) {
    return (
      <Empty title="No such print">
        <Link to={`/s/${slug}`} className="text-accent underline">
          Back to the shop
        </Link>
      </Empty>
    )
  }

  const currency = storeData?.store?.currency ?? 'USD'
  const image = product.images[0]
  const inCart = cart?.items.find((item) => item.productId === product.id)?.quantity ?? 0

  return (
    <div>
      <nav className="mb-8 text-sm text-muted">
        <Link to={`/s/${slug}`} className="transition-colors hover:text-ink">
          ← {storeData?.store?.name ?? 'Back to the shop'}
        </Link>
      </nav>

      <div className="grid gap-10 lg:grid-cols-[1.3fr_1fr] lg:gap-16">
        <div className="overflow-hidden rounded-lg border border-line bg-white">
          {image ? (
            <img
              src={image.url}
              alt={image.altText || product.name}
              className="w-full object-cover"
            />
          ) : (
            <div className="flex aspect-4/5 items-center justify-center text-muted">No photo</div>
          )}
        </div>

        <div className="lg:sticky lg:top-28 lg:self-start">
          <h1 className="text-4xl leading-tight">{product.name}</h1>
          <p className="mt-3 text-2xl tabular-nums text-muted">
            {formatMoney(product.priceCents, currency)}
          </p>

          {product.description && (
            <p className="mt-6 leading-relaxed text-ink/80">{product.description}</p>
          )}

          <div className="mt-8 space-y-3">
            <Button
              className="w-full"
              busy={busy}
              onClick={() => setQuantity(product.id, inCart + 1)}
            >
              {inCart > 0 ? `Add another (${inCart} in basket)` : 'Add to basket'}
            </Button>

            {inCart > 0 && (
              <Link
                to={`/s/${slug}`}
                className="block rounded-md border border-line bg-white px-4 py-2.5 text-center text-sm font-medium transition-colors hover:border-accent hover:bg-accent-soft"
              >
                Go to basket
              </Link>
            )}
          </div>

          {image?.altText && (
            <p className="mt-8 border-t border-line pt-6 text-sm text-muted">
              <span className="font-medium text-ink">In the photograph: </span>
              {image.altText}
            </p>
          )}
        </div>
      </div>
    </div>
  )
}
