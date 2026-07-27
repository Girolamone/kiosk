import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router'
import { Provider as UrqlProvider } from 'urql'
import { createGraphQLClient } from '@kiosk/shared'

import './index.css'
import { App } from './App'

// An empty base URL means same origin: the Vite proxy in development, the Go
// binary serving the built assets in production. Either way the session
// cookie is first-party and rides along on its own.
const client = createGraphQLClient({ baseUrl: '' })

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <UrqlProvider value={client}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </UrqlProvider>
  </StrictMode>,
)
