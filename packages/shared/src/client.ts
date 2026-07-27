import { Client, cacheExchange, fetchExchange } from '@urql/core'

export interface ClientOptions {
  /** Base URL of the API. Empty string means "same origin as this page". */
  baseUrl: string
  /**
   * The web app is served from the same origin as the API, so the session
   * cookie rides along without asking.
   */
  credentials?: RequestCredentials
  /**
   * Called before every request. This is how the two platforms differ: a
   * browser sends a cookie it cannot read, a native app has no cookie jar
   * worth relying on and returns an Authorization header here instead.
   *
   * A function rather than a value, because the token changes when someone
   * signs in or out and the client is built once.
   */
  headers?: () => Record<string, string>
}

export function createGraphQLClient({
  baseUrl,
  credentials = 'same-origin',
  headers,
}: ClientOptions): Client {
  return new Client({
    url: `${baseUrl}/graphql`,
    exchanges: [cacheExchange, fetchExchange],
    fetchOptions: () => ({ credentials, headers: headers?.() ?? {} }),
  })
}

/** What POST /api/uploads returns. Mirrors storage.Object in the Go API. */
export interface UploadedImage {
  key: string
  url: string
  contentType: string
  size: number
}

export class UploadError extends Error {}

/**
 * Uploads one image and returns where it landed. Shared by both apps: the
 * browser hands this a File, React Native hands it a Blob or a
 * `{ uri, name, type }` descriptor, and FormData accepts either.
 */
export async function uploadImage(
  { baseUrl, credentials = 'same-origin', headers }: ClientOptions,
  file: Blob | { uri: string; name: string; type: string },
): Promise<UploadedImage> {
  const form = new FormData()
  form.append('file', file as Blob)

  const response = await fetch(`${baseUrl}/api/uploads`, {
    method: 'POST',
    body: form,
    credentials,
    // Deliberately no Content-Type: the runtime sets it along with the
    // multipart boundary, and overriding it produces a body the server
    // cannot parse.
    headers: headers?.(),
  })

  if (!response.ok) {
    // The API answers with { error } and a message written for a person, so
    // pass it through rather than inventing one from the status code.
    const message = await response
      .json()
      .then((body: { error?: string }) => body.error)
      .catch(() => undefined)
    throw new UploadError(message ?? `Upload failed (${response.status})`)
  }

  return (await response.json()) as UploadedImage
}
