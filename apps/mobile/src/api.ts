import { createGraphQLClient, type ClientOptions } from '@kiosk/shared'
import { authHeaders } from './session'

/**
 * Where the API lives.
 *
 * A phone cannot reach the laptop's localhost, so unlike the web app this
 * always points at a real address. EXPO_PUBLIC_API_URL overrides it for
 * anyone running a server on their own machine and reaching it over the
 * network.
 */
export const API_URL =
  process.env.EXPO_PUBLIC_API_URL ?? 'https://kiosk-257903954523.europe-west3.run.app'

// 'omit' rather than 'include': the app carries its own credential in a
// header, and asking the native fetch to manage cookies as well would mean
// two mechanisms disagreeing about who is signed in.
export const apiOptions: ClientOptions = {
  baseUrl: API_URL,
  credentials: 'omit',
  headers: authHeaders,
}

export const client = createGraphQLClient(apiOptions)
