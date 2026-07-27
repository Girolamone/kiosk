package orders

import "testing"

func TestCartTotal(t *testing.T) {
	cart := Cart{Items: []CartItem{
		{Name: "Speckled Mug", UnitPriceCents: 2800, Quantity: 2},
		{Name: "Ridged Bowl", UnitPriceCents: 4250, Quantity: 1},
	}}

	if got, want := cart.Items[0].LineTotalCents(), 5600; got != want {
		t.Errorf("line total = %d, want %d", got, want)
	}
	// 2800*2 + 4250 = 9850. Computed in integer cents from first to last, so
	// there is no rounding step anywhere for an error to creep into.
	if got, want := cart.TotalCents(), 9850; got != want {
		t.Errorf("cart total = %d, want %d", got, want)
	}
}

func TestEmptyCartTotalsZero(t *testing.T) {
	if got := (Cart{}).TotalCents(); got != 0 {
		t.Errorf("empty cart total = %d, want 0", got)
	}
}

func TestCartTokensAreUnique(t *testing.T) {
	seen := make(map[string]bool, 500)
	for range 500 {
		token, err := NewCartToken()
		if err != nil {
			t.Fatalf("NewCartToken: %v", err)
		}
		if len(token) < 32 {
			t.Fatalf("token %q is too short to be unguessable", token)
		}
		if seen[token] {
			t.Fatalf("NewCartToken returned %q twice", token)
		}
		seen[token] = true
	}
}
