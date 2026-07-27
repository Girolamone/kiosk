# One image serving both the API and the web app, so the deployment is a
# single origin: no CORS, and the session cookie stays first-party.

# ---- build the web app -------------------------------------------------
FROM node:22-alpine AS web
WORKDIR /src

# Manifests first. This layer only changes when a dependency changes, so
# editing source code does not reinstall node_modules on every build.
COPY package.json package-lock.json ./
COPY apps/web/package.json apps/web/
COPY packages/shared/package.json packages/shared/
RUN npm ci

COPY packages/shared packages/shared
COPY apps/web apps/web
RUN npm --workspace web run build

# ---- build the API -----------------------------------------------------
FROM golang:1.26-alpine AS api
WORKDIR /src

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api .
# CGO off so the binary is static and can run on an image with no libc.
# Trimming the build path and symbol table keeps it small and stops absolute
# paths from this machine ending up in the shipped artifact.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# ---- run ---------------------------------------------------------------
# distroless: no shell, no package manager, nothing to exploit if something
# does get in. It carries the CA certificates the binary needs to reach
# Postgres, Stripe, Gemini and Cloud Storage over TLS.
FROM gcr.io/distroless/static-debian12
WORKDIR /app

COPY --from=api /out/server /app/server
COPY --from=web /src/apps/web/dist /app/web

ENV WEB_DIR=/app/web
ENV PORT=8080
EXPOSE 8080

# Runs as a non-root user, provided by the distroless image.
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
