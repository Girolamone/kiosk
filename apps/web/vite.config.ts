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
  server: {
    port: 5173,
    proxy: {
      '/graphql': 'http://localhost:8080',
      '/api': 'http://localhost:8080',
      '/uploads': 'http://localhost:8080',
    },
  },
})
