import { useCallback, useState, type ReactNode } from 'react'
import { useMatch } from 'react-router'
import { useMutation, useQuery } from 'urql'
import { CartDocument, SetCartItemDocument } from '@kiosk/shared'
import { CartContext, readToken, writeToken } from './cart-context'

/**
 * One cart for the whole app rather than one per component.
 *
 * The token lives in state, and state is per instance: with a hook called
 * separately by the header and by the page, adding an item created the cart
 * in one of them and left the other holding a stale null until a reload, so
 * the header showed nothing. A single instance behind a context is what makes
 * the count appear the moment something is added.
 */
export function CartProvider({ children }: { children: ReactNode }) {
  // The basket belongs to one shop, so which shop is being looked at decides
  // which basket this is. end:false so it keeps matching on a product page.
  const inStore = useMatch({ path: '/s/:slug', end: false })
  const storeSlug = inStore?.params.slug ?? ''

  const [token, setToken] = useState(() => readToken(storeSlug))

  // Moving between shops swaps the basket. Done during render rather than in
  // an effect so the first paint after navigating is already correct.
  const [lastSlug, setLastSlug] = useState(storeSlug)
  if (storeSlug !== lastSlug) {
    setLastSlug(storeSlug)
    setToken(readToken(storeSlug))
  }

  const [{ data, fetching }] = useQuery({
    query: CartDocument,
    variables: { token: token ?? '' },
    // urql would otherwise run the query with a placeholder variable.
    pause: !token,
  })

  const [{ fetching: updating }, setCartItem] = useMutation(SetCartItemDocument)

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

  const cart = token ? (data?.cart ?? null) : null

  return (
    <CartContext.Provider
      value={{
        cart,
        loading: fetching && !data,
        busy: updating,
        itemCount: cart?.items.reduce((sum, item) => sum + item.quantity, 0) ?? 0,
        setQuantity,
      }}
    >
      {children}
    </CartContext.Provider>
  )
}
