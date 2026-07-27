import { useCallback } from 'react'
import { useMutation, useQuery } from 'urql'
import { LogInDocument, LogOutDocument, MeDocument, SignUpDocument } from '@kiosk/shared'

/**
 * Who is signed in.
 *
 * There is no token to keep in React state: the session lives in an HttpOnly
 * cookie the browser attaches by itself, and this hook only asks the server
 * who that cookie belongs to. Nothing here can read the credential, which is
 * the point of storing it that way.
 */
export function useSession() {
  const [{ data, fetching, error }, refetch] = useQuery({
    query: MeDocument,
    requestPolicy: 'cache-and-network',
  })

  const reload = useCallback(() => refetch({ requestPolicy: 'network-only' }), [refetch])

  return {
    user: data?.me ?? null,
    loading: fetching && !data,
    error,
    reload,
  }
}

export function useSignUp() {
  return useMutation(SignUpDocument)
}

export function useLogIn() {
  return useMutation(LogInDocument)
}

export function useLogOut() {
  return useMutation(LogOutDocument)
}

/**
 * GraphQL errors arrive wrapped in a CombinedError whose message is prefixed
 * with "[GraphQL] ". The API writes its messages for people to read, so strip
 * the wrapper and show what it actually said.
 */
export function readableError(error: { message: string } | undefined): string | null {
  if (!error) return null
  return error.message.replace(/^\[GraphQL\]\s*/, '')
}
