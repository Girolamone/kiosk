import { useCallback, useEffect, useState } from 'react'
import * as SecureStore from 'expo-secure-store'

const TOKEN_KEY = 'kiosk.session'

/**
 * The token, held where the operating system can protect it.
 *
 * SecureStore is the keychain on iOS and the keystore on Android, so the
 * credential is encrypted at rest and does not sit in a file another app can
 * read. AsyncStorage would have been one line shorter and would have written
 * the session to plain text.
 */
let inMemory: string | null = null

/**
 * Read synchronously by the GraphQL client on every request, which is why
 * there is a copy in memory as well as in the keychain: SecureStore is
 * async, and a request cannot wait on the keychain to build its headers.
 */
export function authHeaders(): Record<string, string> {
  return inMemory ? { Authorization: `Bearer ${inMemory}` } : {}
}

export async function loadStoredToken(): Promise<string | null> {
  try {
    inMemory = await SecureStore.getItemAsync(TOKEN_KEY)
  } catch {
    // A device with no secure hardware, or a simulator in an odd state.
    // Signing in again is a better outcome than refusing to start.
    inMemory = null
  }
  return inMemory
}

export async function storeToken(token: string) {
  inMemory = token
  try {
    await SecureStore.setItemAsync(TOKEN_KEY, token)
  } catch {
    // The session still works for this run; it just will not survive a
    // restart.
  }
}

export async function clearToken() {
  inMemory = null
  try {
    await SecureStore.deleteItemAsync(TOKEN_KEY)
  } catch {
    /* nothing useful to do */
  }
}

export function useStoredSession() {
  const [token, setToken] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadStoredToken().then((stored) => {
      setToken(stored)
      setLoading(false)
    })
  }, [])

  const signIn = useCallback(async (fresh: string) => {
    await storeToken(fresh)
    setToken(fresh)
  }, [])

  const signOut = useCallback(async () => {
    await clearToken()
    setToken(null)
  }, [])

  return { token, loading, signIn, signOut }
}
