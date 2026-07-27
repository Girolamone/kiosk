package catalog

import "testing"

func TestNewStore(t *testing.T) {
	tests := []struct {
		name      string
		storeName string
		slug      string
		wantErr   bool
	}{
		{"accepts a plain slug", "Pine & Salt", "pine-and-salt", false},
		{"lowercases the slug", "Pine & Salt", "Pine-And-Salt", false},
		{"trims surrounding space", "  Pine & Salt  ", "  pine-and-salt  ", false},
		{"rejects an empty name", "", "pine-and-salt", true},
		{"rejects a name that is only space", "   ", "pine-and-salt", true},
		{"rejects a slug with spaces", "Pine & Salt", "pine and salt", true},
		{"rejects a slug with a slash", "Pine & Salt", "pine/salt", true},
		{"rejects a slug that is too short", "Pine & Salt", "ab", true},
		{"rejects a leading hyphen", "Pine & Salt", "-pine", true},
		{"rejects a trailing hyphen", "Pine & Salt", "pine-", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewStore("owner-id", tt.storeName, tt.slug, "")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got store %+v", store)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if store.Slug != "pine-and-salt" {
				t.Errorf("slug = %q, want %q", store.Slug, "pine-and-salt")
			}
			if store.Name != "Pine & Salt" {
				t.Errorf("name = %q, want %q", store.Name, "Pine & Salt")
			}
		})
	}
}

func TestNewProduct(t *testing.T) {
	t.Run("starts as a draft", func(t *testing.T) {
		// Publishing has to be a deliberate second step, or a half-written
		// product appears on the storefront the moment it is created.
		p, err := NewProduct("store-id", "Speckled Mug", "", 2800)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Status != StatusDraft {
			t.Errorf("status = %q, want %q", p.Status, StatusDraft)
		}
	})

	t.Run("accepts a free product", func(t *testing.T) {
		if _, err := NewProduct("store-id", "Sticker", "", 0); err != nil {
			t.Errorf("zero price rejected: %v", err)
		}
	})

	t.Run("rejects a negative price", func(t *testing.T) {
		if _, err := NewProduct("store-id", "Speckled Mug", "", -1); err == nil {
			t.Error("negative price accepted")
		}
	})

	t.Run("rejects an empty name", func(t *testing.T) {
		if _, err := NewProduct("store-id", "   ", "", 2800); err == nil {
			t.Error("blank name accepted")
		}
	})
}
