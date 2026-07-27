import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// The dev server proxies the API instead of pointing the app at
// http://localhost:8080 directly. That keeps development same-origin, which
// matters here: the session is an HttpOnly cookie, and a cross-origin setup
// would need CORS with credentials and SameSite=None in dev but not in
// production. Proxying makes both behave the same way.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    // Force one React into the bundle.
    //
    // In a workspace, a second app pinning a different React version makes
    // npm nest one copy and hoist another. A dependency resolving the
    // hoisted copy then calls hooks from a React that is not the one
    // rendering, and every hook reads a null dispatcher: "Cannot read
    // properties of null (reading 'useContext')", with a blank page and
    // nothing in the console during a normal load.
    //
    // Dev is more forgiving about this than the production build, so it is
    // the kind of fault that only appears after deploying.
    dedupe: ['react', 'react-dom'],
  },
  server: {
    port: 5173,
    proxy: {
      '/graphql': 'http://localhost:8080',
      '/api': 'http://localhost:8080',
      '/uploads': 'http://localhost:8080',
    },
  },
})
