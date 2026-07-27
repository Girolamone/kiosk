import { createContext, useContext } from 'react'
import type { CartContentsFragment } from '@kiosk/shared'

export interface CartState {
  cart: CartContentsFragment | null
  loading: boolean
  busy: boolean
  itemCount: number
  setQuantity: (productId: string, quantity: number) => Promise<unknown>
}

export const CartContext = createContext<CartState | null>(null)

export function useCart(): CartState {
  const state = useContext(CartContext)
  if (!state) throw new Error('useCart must be used inside a CartProvider')
  return state
}

/**
 * The cart token is kept per shop.
 *
 * A cart belongs to one store on the server, so one token in localStorage
 * would mean visiting a second shop silently abandons the first basket.
 */
function storageKey(storeSlug: string) {
  return `kiosk.cart.${storeSlug}`
}

export function readToken(storeSlug: string): string | null {
  if (!storeSlug) return null
  try {
    return localStorage.getItem(storageKey(storeSlug))
  } catch {
    // Private browsing modes throw rather than return null. A shopper with
    // no storage gets a basket that lasts the page, which beats a crash.
    return null
  }
}

export function writeToken(storeSlug: string, token: string) {
  try {
    localStorage.setItem(storageKey(storeSlug), token)
  } catch {
    /* see readToken */
  }
}
