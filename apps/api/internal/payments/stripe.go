// Package payments takes a cart to Stripe Checkout and reads back what
// happened.
package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"
)

// ErrUnavailable means payments are not configured. Like copy generation,
// this is something the application can be missing without falling over: the
// shop still works, it just cannot take money.
var ErrUnavailable = errors.New("payments are not configured")

// Session is a checkout in progress.
type Session struct {
	ID  string
	URL string
}

// Line is one thing being bought. Name and price come from the database, not
// from the browser.
type Line struct {
	Name           string
	UnitPriceCents int
	Quantity       int
}

// Paid describes a checkout that has been paid for. A zero value means the
// event was not a completed payment and there is nothing to do.
type Paid struct {
	SessionID string
	// Reference is whatever was passed as SessionRequest.Reference.
	Reference string
}

func (p Paid) IsZero() bool { return p.SessionID == "" }

type Gateway interface {
	CreateSession(ctx context.Context, req SessionRequest) (Session, error)
	// ParseWebhook verifies the signature and reports a completed payment.
	ParseWebhook(payload []byte, signature string) (Paid, error)
}

type SessionRequest struct {
	Email      string
	Currency   string
	Lines      []Line
	SuccessURL string
	CancelURL  string
	// Reference ties the Stripe session back to something of ours, and shows
	// up in the Stripe dashboard.
	Reference string
}

type Stripe struct {
	client        *stripe.Client
	webhookSecret string
}

func NewStripe(secretKey, webhookSecret string) *Stripe {
	// A client instance rather than the package-level stripe.Key, so the
	// credential is owned by this object instead of by global state.
	return &Stripe{
		client:        stripe.NewClient(secretKey),
		webhookSecret: webhookSecret,
	}
}

func (s *Stripe) CreateSession(ctx context.Context, req SessionRequest) (Session, error) {
	if len(req.Lines) == 0 {
		return Session{}, errors.New("nothing to charge for")
	}

	lineItems := make([]*stripe.CheckoutSessionCreateLineItemParams, 0, len(req.Lines))
	for _, line := range req.Lines {
		lineItems = append(lineItems, &stripe.CheckoutSessionCreateLineItemParams{
			Quantity: stripe.Int64(int64(line.Quantity)),
			PriceData: &stripe.CheckoutSessionCreateLineItemPriceDataParams{
				Currency: stripe.String(strings.ToLower(req.Currency)),
				// The amount is already in minor units, which is what Stripe
				// wants. No division happens anywhere along this path.
				UnitAmount: stripe.Int64(int64(line.UnitPriceCents)),
				ProductData: &stripe.CheckoutSessionCreateLineItemPriceDataProductDataParams{
					Name: stripe.String(line.Name),
				},
			},
		})
	}

	params := &stripe.CheckoutSessionCreateParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems:         lineItems,
		SuccessURL:        stripe.String(req.SuccessURL),
		CancelURL:         stripe.String(req.CancelURL),
		ClientReferenceID: stripe.String(req.Reference),
	}
	if req.Email != "" {
		params.CustomerEmail = stripe.String(req.Email)
	}

	created, err := s.client.V1CheckoutSessions.Create(ctx, params)
	if err != nil {
		return Session{}, fmt.Errorf("create checkout session: %w", err)
	}
	return Session{ID: created.ID, URL: created.URL}, nil
}

// ParseWebhook verifies that the request really came from Stripe.
//
// Without the signature check this endpoint is a public "mark my order paid"
// button: it is unauthenticated by design, because Stripe is the caller.
func (s *Stripe) ParseWebhook(payload []byte, signature string) (Paid, error) {
	if s.webhookSecret == "" {
		return Paid{}, errors.New("no webhook secret configured, refusing to trust the payload")
	}

	event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		return Paid{}, fmt.Errorf("verify webhook signature: %w", err)
	}

	if event.Type != "checkout.session.completed" {
		// Stripe sends plenty of events nobody here subscribed to caring
		// about. Not an error, just nothing to do.
		return Paid{}, nil
	}

	var session struct {
		ID                string `json:"id"`
		PaymentStatus     string `json:"payment_status"`
		ClientReferenceID string `json:"client_reference_id"`
	}
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return Paid{}, fmt.Errorf("decode checkout session: %w", err)
	}

	// "completed" only means the customer finished the flow. Bank transfers
	// and other delayed methods complete while still unpaid.
	if session.PaymentStatus != "paid" {
		return Paid{}, nil
	}
	return Paid{SessionID: session.ID, Reference: session.ClientReferenceID}, nil
}

// Disabled stands in when Stripe is not configured.
type Disabled struct{}

func (Disabled) CreateSession(context.Context, SessionRequest) (Session, error) {
	return Session{}, ErrUnavailable
}

func (Disabled) ParseWebhook([]byte, string) (Paid, error) {
	return Paid{}, ErrUnavailable
}
