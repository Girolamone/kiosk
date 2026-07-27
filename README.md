# Kiosk

A small storefront platform: sellers create a store, list products, and take orders.
Product copy is generated from the product photo.

Go + GraphQL + Postgres on the back, React on the web, Expo on mobile, one shared
schema between them.

**Status:** in development.

## Layout

```
apps/api      Go GraphQL API
apps/web      React storefront + seller dashboard
apps/mobile   Expo seller app
packages/     Code shared between web and mobile
```

## Running the API

```sh
cp .env.example .env   # fill in DATABASE_URL
cd apps/api
go run ./cmd/server
```
