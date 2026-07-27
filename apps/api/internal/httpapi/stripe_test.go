package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Girolamone/kiosk/apps/api/internal/orders"
	"github.com/Girolamone/kiosk/apps/api/internal/payments"
)

const testWebhookSecret = "whsec_testsecret"

// signedRequest builds a webhook request the way Stripe does: an HMAC of
// "<timestamp>.<payload>" keyed with the endpoint secret. Constructing it
// here rather than mocking the verifier means the real signature check runs.
func signedRequest(t *testing.T, secret, payload string) *http.Request {
	t.Helper()

	timestamp := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.%s", timestamp, payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/api/stripe/webhook", strings.NewReader(payload))
	req.Header.Set("Stripe-Signature", fmt.Sprintf("t=%d,v1=%s", timestamp, signature))
	return req
}

// The top-level "object": "event" is not decoration: stripe-go uses it to
// tell a full event from the thin EventNotification format, and refuses the
// payload without it.
func completedEvent(sessionID, paymentStatus, reference string) string {
	return fmt.Sprintf(`{
		"id": "evt_test",
		"object": "event",
		"api_version": "2020-08-27",
		"type": "checkout.session.completed",
		"data": {"object": {
			"id": %q,
			"object": "checkout.session",
			"payment_status": %q,
			"client_reference_id": %q
		}}
	}`, sessionID, paymentStatus, reference)
}

// fakeSettler records what the handler asked it to do.
type fakeSettler struct {
	markPaidCalls []string
	alreadyPaid   bool
	cleared       []string
	markPaidErr   error
}

func (f *fakeSettler) MarkPaid(_ context.Context, sessionID string) (bool, error) {
	f.markPaidCalls = append(f.markPaidCalls, sessionID)
	if f.markPaidErr != nil {
		return false, f.markPaidErr
	}
	return !f.alreadyPaid, nil
}

func (f *fakeSettler) CartByToken(_ context.Context, token string) (orders.Cart, error) {
	if token == "" {
		return orders.Cart{}, orders.ErrNotFound
	}
	return orders.Cart{ID: "cart-" + token, Token: token}, nil
}

func (f *fakeSettler) ClearCart(_ context.Context, cartID string) error {
	f.cleared = append(f.cleared, cartID)
	return nil
}

func newHandler(settler OrderSettler) http.Handler {
	gateway := payments.NewStripe("sk_test_unused", testWebhookSecret)
	return StripeWebhook(gateway, settler, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestWebhookSettlesAPaidCheckout(t *testing.T) {
	settler := &fakeSettler{}
	recorder := httptest.NewRecorder()

	newHandler(settler).ServeHTTP(recorder,
		signedRequest(t, testWebhookSecret, completedEvent("cs_test_123", "paid", "carttoken")))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body: %s", recorder.Code, recorder.Body)
	}
	if len(settler.markPaidCalls) != 1 || settler.markPaidCalls[0] != "cs_test_123" {
		t.Errorf("MarkPaid calls = %v, want one call for cs_test_123", settler.markPaidCalls)
	}
	if len(settler.cleared) != 1 || settler.cleared[0] != "cart-carttoken" {
		t.Errorf("cleared = %v, want the paid cart emptied", settler.cleared)
	}
}

// This is the whole reason the endpoint can be public. Without it, anyone
// could POST a payment notification and mark their own order paid.
func TestWebhookRejectsAForgedSignature(t *testing.T) {
	settler := &fakeSettler{}
	recorder := httptest.NewRecorder()

	newHandler(settler).ServeHTTP(recorder,
		signedRequest(t, "whsec_theattackersguess", completedEvent("cs_test_123", "paid", "")))

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", recorder.Code)
	}
	if len(settler.markPaidCalls) != 0 {
		t.Errorf("a forged webhook settled an order: %v", settler.markPaidCalls)
	}
}

func TestWebhookRejectsAMissingSignature(t *testing.T) {
	settler := &fakeSettler{}
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/stripe/webhook",
		strings.NewReader(completedEvent("cs_test_123", "paid", "")))

	newHandler(settler).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", recorder.Code)
	}
	if len(settler.markPaidCalls) != 0 {
		t.Errorf("an unsigned webhook settled an order: %v", settler.markPaidCalls)
	}
}

// Stripe delivers at least once and retries. A repeat has to be acknowledged
// with a 2xx, or Stripe keeps redelivering forever.
func TestWebhookAcknowledgesARepeat(t *testing.T) {
	settler := &fakeSettler{alreadyPaid: true}
	recorder := httptest.NewRecorder()

	newHandler(settler).ServeHTTP(recorder,
		signedRequest(t, testWebhookSecret, completedEvent("cs_test_123", "paid", "carttoken")))

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 so Stripe stops retrying", recorder.Code)
	}
	if len(settler.cleared) != 0 {
		t.Errorf("a repeat delivery touched the cart again: %v", settler.cleared)
	}
}

// A completed checkout is not necessarily a paid one: delayed payment methods
// finish the flow and settle later.
func TestWebhookIgnoresAnUnpaidCheckout(t *testing.T) {
	settler := &fakeSettler{}
	recorder := httptest.NewRecorder()

	newHandler(settler).ServeHTTP(recorder,
		signedRequest(t, testWebhookSecret, completedEvent("cs_test_123", "unpaid", "carttoken")))

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", recorder.Code)
	}
	if len(settler.markPaidCalls) != 0 {
		t.Errorf("an unpaid checkout was settled: %v", settler.markPaidCalls)
	}
}

func TestWebhookIgnoresOtherEventTypes(t *testing.T) {
	settler := &fakeSettler{}
	recorder := httptest.NewRecorder()
	payload := `{"id":"evt_test","object":"event","api_version":"2020-08-27",
		"type":"customer.created","data":{"object":{"id":"cus_1"}}}`

	newHandler(settler).ServeHTTP(recorder, signedRequest(t, testWebhookSecret, payload))

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", recorder.Code)
	}
	if len(settler.markPaidCalls) != 0 {
		t.Errorf("an unrelated event settled an order: %v", settler.markPaidCalls)
	}
}

// A database failure must not be acknowledged: the order is still pending, so
// Stripe retrying is exactly the behaviour wanted.
func TestWebhookAsksForARetryWhenSettlementFails(t *testing.T) {
	settler := &fakeSettler{markPaidErr: context.DeadlineExceeded}
	recorder := httptest.NewRecorder()

	newHandler(settler).ServeHTTP(recorder,
		signedRequest(t, testWebhookSecret, completedEvent("cs_test_123", "paid", "carttoken")))

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 so Stripe retries", recorder.Code)
	}
}
