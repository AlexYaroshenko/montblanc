package parser

import (
	"testing"
)

func TestParseRefugeContent(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		refuge    Refuge
		wantErr   bool
		checkFunc func(t *testing.T, refuge Refuge)
	}{
		{
			name: "successful parse with available and full dates",
			html: `
				<div class="day dispo">08/03<span>2</span></div>
				<div class="day dispo">08/08<span>1</span></div>
				<div class="day complet">08/10</div>
			`,
			refuge: Refuge{
				Name:  "Test Refuge",
				Dates: make(map[string]string),
			},
			wantErr: false,
			checkFunc: func(t *testing.T, refuge Refuge) {
				if len(refuge.Dates) != 3 {
					t.Errorf("expected 3 dates, got %d", len(refuge.Dates))
				}
				if refuge.Dates["2025-08-03"] != "2" {
					t.Errorf("expected 2 places for 2025-08-03, got %s", refuge.Dates["2025-08-03"])
				}
				if refuge.Dates["2025-08-08"] != "1" {
					t.Errorf("expected 1 place for 2025-08-08, got %s", refuge.Dates["2025-08-08"])
				}
				if refuge.Dates["2025-08-10"] != "Full" {
					t.Errorf("expected Full for 2025-08-10, got %s", refuge.Dates["2025-08-10"])
				}
			},
		},
		{
			name: "only available dates",
			html: `
				<div class="day dispo">07/15<span>3</span></div>
				<div class="day dispo">07/20<span>1</span></div>
			`,
			refuge: Refuge{
				Name:  "Test Refuge",
				Dates: make(map[string]string),
			},
			wantErr: false,
			checkFunc: func(t *testing.T, refuge Refuge) {
				if len(refuge.Dates) != 2 {
					t.Errorf("expected 2 dates, got %d", len(refuge.Dates))
				}
				if refuge.Dates["2025-07-15"] != "3" {
					t.Errorf("expected 3 places for 2025-07-15, got %s", refuge.Dates["2025-07-15"])
				}
				if refuge.Dates["2025-07-20"] != "1" {
					t.Errorf("expected 1 place for 2025-07-20, got %s", refuge.Dates["2025-07-20"])
				}
			},
		},
		{
			name: "only full dates",
			html: `
				<div class="day complet">07/01</div>
				<div class="day complet">07/02</div>
			`,
			refuge: Refuge{
				Name:  "Test Refuge",
				Dates: make(map[string]string),
			},
			wantErr: false,
			checkFunc: func(t *testing.T, refuge Refuge) {
				if len(refuge.Dates) != 2 {
					t.Errorf("expected 2 dates, got %d", len(refuge.Dates))
				}
				if refuge.Dates["2025-07-01"] != "Full" {
					t.Errorf("expected Full for 2025-07-01, got %s", refuge.Dates["2025-07-01"])
				}
				if refuge.Dates["2025-07-02"] != "Full" {
					t.Errorf("expected Full for 2025-07-02, got %s", refuge.Dates["2025-07-02"])
				}
			},
		},
		{
			name: "invalid HTML",
			html: `<invalid>html</invalid>`,
			refuge: Refuge{
				Name:  "Test Refuge",
				Dates: make(map[string]string),
			},
			wantErr: false,
			checkFunc: func(t *testing.T, refuge Refuge) {
				if len(refuge.Dates) != 0 {
					t.Errorf("expected 0 dates for invalid HTML, got %d", len(refuge.Dates))
				}
			},
		},
		{
			name: "empty HTML",
			html: "",
			refuge: Refuge{
				Name:  "Test Refuge",
				Dates: make(map[string]string),
			},
			wantErr: false,
			checkFunc: func(t *testing.T, refuge Refuge) {
				if len(refuge.Dates) != 0 {
					t.Errorf("expected 0 dates for empty HTML, got %d", len(refuge.Dates))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseRefugeContent(tt.html, &tt.refuge)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRefugeContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Run additional checks if provided
			if tt.checkFunc != nil {
				tt.checkFunc(t, tt.refuge)
			}
		})
	}
}
