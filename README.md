# Kiosk

[![API](https://github.com/Girolamone/kiosk/actions/workflows/api.yml/badge.svg)](https://github.com/Girolamone/kiosk/actions/workflows/api.yml)
[![Web](https://github.com/Girolamone/kiosk/actions/workflows/web.yml/badge.svg)](https://github.com/Girolamone/kiosk/actions/workflows/web.yml)

A small storefront platform. A seller photographs the thing they are selling;
Kiosk reads the photograph and drafts the title, description and alt text;
they correct what it got wrong and publish.

**Live: https://kiosk-257903954523.europe-west3.run.app**

- Shop: [/s/northlight-press](https://kiosk-257903954523.europe-west3.run.app/s/northlight-press)
- Seller dashboard: sign in as `demo@kiosk.dev` / `kioskdemo2026`
- GraphQL playground: [/playground](https://kiosk-257903954523.europe-west3.run.app/playground)

Checkout runs in Stripe test mode. Card `4242 4242 4242 4242`, any future
expiry, any CVC.

Go, GraphQL and Postgres on the back; React on the front; one binary serving
both. Around 3,500 lines of hand-written Go and 1,200 of TypeScript.

---

## What it does

**Public storefront.** Each shop lives at `/s/<slug>` with its published
products, a basket, and Stripe Checkout.

**Seller dashboard.** Create a shop, add products, publish and unpublish.
Drafts are visible only to the owner.

**Listing copy from a photograph.** Upload a product photo and the title,
description and alt text come back written. Every field stays editable, and
the form still works when the model is unavailable — which is the point of
the next section.

**A seller's app for iOS and Android.** The same feature, but the photograph
comes from the camera you are holding rather than a file you have to find
first. Built on the same schema and the same generated client as the web app.

---

## Decisions worth reading

The parts of this that were actual decisions rather than typing.

### Money is integer cents, start to finish

`price_cents integer`, never a float. `0.1 + 0.2` is not `0.3` in binary
floating point, and the error compounds across a basket until the total shown
and the total charged differ by a cent. Stripe takes minor units too, so the
value travels from Postgres to the payment page without ever being divided.
The only division is in `formatMoney`, at the moment of display.

### An order remembers what was bought

`order_items` copies the product name and unit price at purchase time rather
than pointing at the product. If a seller renames something or raises a price,
yesterday's order still says what the customer actually bought and paid. A
foreign key alone would rewrite history retroactively, which is wrong and in
most of Europe illegal.

### The AI feature is allowed to be absent

Copy generation sits behind a `CopyGenerator` interface with a `Disabled`
implementation. With no API key the server logs a warning, starts anyway, and
the seller writes the listing themselves. Verified by running with the key
removed: uploads and the storefront are untouched, only that one field
declines.

It is an accelerator, not a dependency. The same is true of Stripe: a
deployment with no keys still runs a shop, it just cannot take money.

### Sessions are cookies the page cannot read

The session is a signed token in an `HttpOnly` cookie, so a cross-site
scripting bug cannot walk off with it. Confirmed from inside the running page:
`document.cookie` is empty while the session works.

Log in answers the same way whether the email is unknown or the password is
wrong, and hashes a dummy value in the unknown-email case. Returning early
there would make unregistered addresses measurably faster to check, which is
enough to enumerate users with a stopwatch.

### Development and production are the same shape

The Vite dev server proxies `/graphql`, `/api` and `/uploads` to the Go
process, and in production one binary serves both the API and the built app.
Both are single-origin, so there is no CORS anywhere and the cookie is
first-party in both. Pointing the dev app straight at `:8080` would have meant
`SameSite=None` and credentialed CORS in development but not in production —
the kind of difference that only surfaces after a deploy.

### One schema, generated into the client

`packages/shared` holds types generated from the Go SDL files, the typed
operations, and the urql client. Codegen reads the `.graphqls` files rather
than a running server, so types regenerate offline and CI verifies they match
without standing up a database. Rename a field in Go and TypeScript stops
compiling.

The package imports neither React nor the DOM, which is what lets the mobile
app depend on it unchanged.

### An N+1 that was measured, not guessed

Listing a storefront called the images resolver once per product. On a shop
with twelve products that was 14 queries. A request-scoped dataloader batches
them into one `ANY($1)` lookup: 3 queries, and a thirteenth product adds none.

Loaders are built per request by middleware, not once per process. That is not
a detail — a loader caches what it fetched, so a shared one would serve one
user data fetched for another and never notice a write. The cache is only safe
because it dies with the request.

There is an SQL tracer behind `LOG_SQL=true` that made the problem visible.
An N+1 is invisible in a profiler that reports total time and obvious when the
same statement scrolls past twelve times.

### The webhook is public, so the signature is everything

`POST /api/stripe/webhook` is unauthenticated because Stripe is the caller.
The signature check is the only thing between it and anyone on the internet
marking their own order paid, so an unverified payload is refused rather than
trusted.

Settlement is idempotent through a status check in the `UPDATE`: Stripe
delivers at least once and retries, so the second delivery has to change
nothing. A database failure answers 500 on purpose — the order is still
pending, so a retry is exactly what should happen.

Tests sign payloads the way Stripe signs them rather than stubbing the
verifier, and the forged-signature case was confirmed to fail when the handler
is pointed at the attacker's key. A test that passes when the thing it guards
is removed is not a test.

### Uploads are not trusted

The browser's `Content-Type` is ignored and the file is identified by its
bytes. An HTML file named `.png` is refused with a 415 and never written,
because storing it would mean serving it back from this domain for a browser
to execute. The uploaded filename is discarded too — keys are random hex plus
an extension derived from the detected type.

---

## What is deliberately not here

Being explicit about this is more useful than pretending otherwise.

- **The mobile app is not on a store.** It runs through Expo Go from source,
  and has been bundled but not shipped. What is shared between it and the web
  app is the schema, the generated client and the business rules — not the
  interface. Nobody writes one interface for both, and a README claiming
  otherwise would be contradicted by two component folders.
- **Orders are recorded, not fulfilled.** There is no seller order screen,
  no email, no shipping.
- **Secrets are environment variables on Cloud Run,** not Secret Manager.
  Fine for a demo with one operator; not what I would ship to a team.
- **No rate limiting.** The copy generation endpoint costs quota and is
  behind a login, but nothing stops a signed-in user from calling it in a
  loop.
- **One image per product.** The schema supports more; the UI does not.
- **Test coverage is uneven.** 24 tests, concentrated where being wrong is
  expensive: token handling, webhook signatures, cart totals, routing,
  validation. The resolvers are covered by having been run, not by tests.

---

## Running it

Needs Go 1.26, Node 22, and a Postgres connection string. Nothing else — the
default storage driver writes to local disk, so no cloud account is required.

```sh
cp .env.example .env        # fill in DATABASE_URL and JWT_SECRET
npm install

cd apps/api
go run ./cmd/migrate up
go run ./cmd/seed           # optional: a demo shop with products
go run ./cmd/server
```

```sh
cd apps/web
npm run dev                 # http://localhost:5173
```

`GEMINI_API_KEY` enables copy generation, `STRIPE_SECRET_KEY` enables
checkout. Without either, everything else still works.

### Layout

```
apps/api          Go: GraphQL API, migrations, seed
  graph/            schema (.graphqls) and resolvers
  internal/         account, ai, auth, catalog, db, httpapi,
                    loaders, orders, payments, storage
apps/web          React: storefront and seller dashboard
apps/mobile       Expo: the seller's app, iOS and Android
packages/shared   generated types, operations, GraphQL client
```

### The mobile app

```sh
cd apps/mobile
npx expo start          # scan the QR code with Expo Go
```

It talks to the deployed API by default; `EXPO_PUBLIC_API_URL` points it
somewhere else. Sign in with the demo account above.

Authentication differs from the web on purpose. A browser is handed a token
in a cookie it cannot read, which is what stops a cross-site scripting bug
stealing the session. A native app has no such protection to lose and no
cookie jar worth relying on, so it asks for the token directly, keeps it in
the keychain, and sends it as a bearer header. Same token, same verification;
only the way it travels differs.

### Deployment

One container image built from the repo root: Node builds the web app, Go
builds a static binary, and a distroless image carries both. Cloud Run in
`europe-west3`, Postgres on Neon in the same region, product photos in Cloud
Storage.

`max-instances=1` is set deliberately. It caps what the service can cost under
traffic it was never meant to take.

---

## Built with Claude Code

Written over one working session, pair-programming with Claude Code — the
architecture decisions, the bugs and the corrections are all in the commit
history, including the ones that were wrong the first time.
