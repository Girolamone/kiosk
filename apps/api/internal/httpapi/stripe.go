package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/Girolamone/kiosk/apps/api/internal/orders"
	"github.com/Girolamone/kiosk/apps/api/internal/payments"
)

// maxWebhookBytes bounds the request. Stripe's payloads are small; anything
// larger is not from Stripe.
const maxWebhookBytes = 1 << 20 // 1 MiB

// OrderSettler is the slice of the orders repository this endpoint needs.
// Naming it here rather than taking the concrete type means the handler can
// be tested against a fake, which is the difference between the retry and
// idempotency paths being covered and being hoped about.
type OrderSettler interface {
	MarkPaid(ctx context.Context, sessionID string) (bool, error)
	CartByToken(ctx context.Context, token string) (orders.Cart, error)
	ClearCart(ctx context.Context, cartID string) error
}

// StripeWebhook settles orders when Stripe reports a payment.
//
// This endpoint is deliberately unauthenticated, because Stripe is the
// caller. The signature check inside ParseWebhook is the only thing standing
// between it and anyone on the internet marking their own order paid, so an
// unverified payload is refused rather than trusted.
func StripeWebhook(gateway payments.Gateway, repo OrderSettler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBytes))
		if err != nil {
			writeError(w, http.StatusBadRequest, "could not read the payload")
			return
		}

		paid, err := gateway.ParseWebhook(payload, r.Header.Get("Stripe-Signature"))
		if err != nil {
			logger.Warn("rejected a stripe webhook", "err", err)
			writeError(w, http.StatusBadRequest, "signature check failed")
			return
		}
		if paid.IsZero() {
			// Some other event, or a checkout that completed without being
			// paid. Acknowledge it: a non-2xx would make Stripe retry
			// something there is no point retrying.
			w.WriteHeader(http.StatusOK)
			return
		}

		settled, err := repo.MarkPaid(r.Context(), paid.SessionID)
		if err != nil {
			// Do not acknowledge. Stripe retries, and the order is still
			// pending, so the retry is exactly what should happen.
			logger.Error("could not settle order", "session", paid.SessionID, "err", err)
			writeError(w, http.StatusInternalServerError, "could not settle the order")
			return
		}

		if !settled {
			// Already paid. Stripe delivers at least once and retries, so
			// seeing the same event twice is normal, not a problem.
			logger.Info("stripe webhook was a repeat", "session", paid.SessionID)
			w.WriteHeader(http.StatusOK)
			return
		}

		logger.Info("order paid", "session", paid.SessionID)

		// Emptying the basket is a courtesy, not part of the payment. If it
		// fails the money is still taken and the order is still recorded, so
		// log it and acknowledge anyway.
		if paid.Reference != "" {
			if cart, err := repo.CartByToken(r.Context(), paid.Reference); err == nil {
				if err := repo.ClearCart(r.Context(), cart.ID); err != nil {
					logger.Warn("could not clear the paid cart", "err", err)
				}
			}
		}

		w.WriteHeader(http.StatusOK)
	})
}
