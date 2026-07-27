import { useState } from 'react'
import { useMutation } from 'urql'
import { CreateCheckoutDocument, formatMoney, type CartContentsFragment } from '@kiosk/shared'
import { Button, Card, ErrorNote, Field, Input } from './ui'
import { readableError } from '../lib/session'

export function CartPanel({
  cart,
  busy,
  onSetQuantity,
}: {
  cart: CartContentsFragment
  busy: boolean
  onSetQuantity: (productId: string, quantity: number) => void
}) {
  const [email, setEmail] = useState('')
  const [{ fetching, error }, createCheckout] = useMutation(CreateCheckoutDocument)

  async function checkout(event: React.FormEvent) {
    event.preventDefault()
    const result = await createCheckout({ token: cart.token, email })
    const url = result.data?.createCheckout.url
    if (url) {
      // Stripe hosts the payment page; hand the whole browser over to it.
      window.location.href = url
    }
  }

  return (
    <Card className="mb-10 p-5">
      <h2 className="text-2xl">Your basket</h2>

      <ul className="mt-4 divide-y divide-line">
        {cart.items.map((item) => (
          <li key={item.productId} className="flex items-center gap-4 py-3">
            <div className="min-w-0 flex-1">
              <p className="truncate">{item.name}</p>
              <p className="text-sm text-muted">
                {formatMoney(item.unitPriceCents, cart.currency)} each
              </p>
            </div>

            <div className="flex items-center gap-2">
              <Button
                variant="secondary"
                aria-label={`One fewer ${item.name}`}
                disabled={busy}
                onClick={() => onSetQuantity(item.productId, item.quantity - 1)}
              >
                −
              </Button>
              <span className="w-6 text-center tabular-nums">{item.quantity}</span>
              <Button
                variant="secondary"
                aria-label={`One more ${item.name}`}
                disabled={busy || item.quantity >= 99}
                onClick={() => onSetQuantity(item.productId, item.quantity + 1)}
              >
                +
              </Button>
            </div>

            <span className="w-20 shrink-0 text-right tabular-nums">
              {formatMoney(item.lineTotalCents, cart.currency)}
            </span>
          </li>
        ))}
      </ul>

      <div className="mt-4 flex items-baseline justify-between border-t border-line pt-4">
        <span className="font-medium">Total</span>
        <span className="text-xl tabular-nums">
          {formatMoney(cart.totalCents, cart.currency)}
        </span>
      </div>

      <form onSubmit={checkout} className="mt-5 flex flex-wrap items-end gap-3">
        <div className="min-w-56 flex-1">
          <Field label="Email for the receipt">
            <Input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              required
            />
          </Field>
        </div>
        <Button type="submit" busy={fetching}>
          Checkout
        </Button>
      </form>

      <p className="mt-3 text-xs text-muted">
        Stripe test mode. Card 4242 4242 4242 4242, any future expiry, any CVC.
      </p>

      <div className="mt-3">
        <ErrorNote>{readableError(error)}</ErrorNote>
      </div>
    </Card>
  )
}
