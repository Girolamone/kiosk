import { useCallback, useState } from 'react'
import { useMutation, useQuery } from 'urql'
import { CartDocument, SetCartItemDocument, type CartContentsFragment } from '@kiosk/shared'

/**
 * The cart token is kept per shop.
 *
 * A cart belongs to one store on the server, so one token in localStorage
 * would mean visiting a second shop silently abandons the first basket.
 */
function storageKey(storeSlug: string) {
  return `kiosk.cart.${storeSlug}`
}

function readToken(storeSlug: string): string | null {
  try {
    return localStorage.getItem(storageKey(storeSlug))
  } catch {
    // Private browsing modes throw rather than return null. A shopper with
    // no storage gets a basket that lasts the page, which beats a crash.
    return null
  }
}

function writeToken(storeSlug: string, token: string) {
  try {
    localStorage.setItem(storageKey(storeSlug), token)
  } catch {
    /* see readToken */
  }
}

export function useCart(storeSlug: string) {
  const [token, setToken] = useState(() => readToken(storeSlug))

  const [{ data, fetching }] = useQuery({
    query: CartDocument,
    variables: { token: token ?? '' },
    // urql still runs a query with a placeholder variable, so skip it
    // outright when there is no basket to ask about.
    pause: !token,
  })

  const [{ fetching: updating, error }, setCartItem] = useMutation(SetCartItemDocument)

  const setQuantity = useCallback(
    async (productId: string, quantity: number) => {
      const result = await setCartItem({
        input: { storeSlug, productId, quantity, token },
      })
      const updated = result.data?.setCartItem
      if (updated && updated.token !== token) {
        // The server started a new basket for us; remember which one.
        setToken(updated.token)
        writeToken(storeSlug, updated.token)
      }
      return result
    },
    [setCartItem, storeSlug, token],
  )

  const cart: CartContentsFragment | null = data?.cart ?? null

  return {
    cart,
    token,
    loading: fetching && !data,
    busy: updating,
    error,
    setQuantity,
    itemCount: cart?.items.reduce((sum, item) => sum + item.quantity, 0) ?? 0,
  }
}
