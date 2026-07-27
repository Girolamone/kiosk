package graph

import (
	"github.com/Girolamone/kiosk/apps/api/internal/account"
	"github.com/Girolamone/kiosk/apps/api/internal/ai"
	"github.com/Girolamone/kiosk/apps/api/internal/auth"
	"github.com/Girolamone/kiosk/apps/api/internal/catalog"
	"github.com/Girolamone/kiosk/apps/api/internal/storage"
)

//go:generate go tool gqlgen generate

// Resolver is the dependency root for every resolver. Anything a resolver
// needs is a field here, injected once at startup.
type Resolver struct {
	Catalog    *catalog.Repository
	Accounts   *account.Service
	Tokens     *auth.TokenIssuer
	Files      storage.Store
	Copywriter ai.CopyGenerator
}
