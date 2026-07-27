package graph

import "github.com/Girolamone/kiosk/apps/api/internal/catalog"

//go:generate go tool gqlgen generate

// Resolver is the dependency root for every resolver. Anything a resolver
// needs is a field here, injected once at startup.
type Resolver struct {
	Catalog *catalog.Repository
}
