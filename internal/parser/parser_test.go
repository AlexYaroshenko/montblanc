package parser

import (
    "os"
    "testing"
    "time"
)

func TestParseRefugeContentFromFile(t *testing.T) {
	content, err := os.ReadFile("response.html")
	if err != nil {
		t.Fatalf("failed to read response.html: %v", err)
	}

	refuge := Refuge{
		Name:  "Test Refuge",
		Dates: make(map[string]string),
	}
    err = parseRefugeContent(string(content), &refuge, time.Date(2025, 8, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("failed to parse response.html: %v", err)
	}

	// Check if the dates are parsed correctly
	if len(refuge.Dates) != 62 {
		t.Errorf("expected 62 dates, got %d", len(refuge.Dates))
	}
	if refuge.Dates["2025-08-03"] != "2" {
		t.Errorf("expected 2 places for 2025-08-03, got %s", refuge.Dates["2025-08-03"])
	}
	if refuge.Dates["2025-08-25"] != "1" {
		t.Errorf("expected 1 places for 2025-08-08, got %s", refuge.Dates["2025-08-25"])
	}
}

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
				<div class="day dispo">
					<a href="#" data-date="2025-08-03" id="date20250803" onclick="return false;">
						<span class="date">08/03</span>
						<span class="place">2</span>
					</a>
				</div>
				<div class="day dispo">
					<a href="#" data-date="2025-08-08" id="date20250808" onclick="return false;">
						<span class="date">08/08</span>
						<span class="place">1</span>
					</a>
				</div>
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
				<div class="day dispo">
					<a href="#" data-date="2025-07-15" id="date20250715" onclick="return false;">
						<span class="date">07/15</span>
						<span class="place">3</span>
					</a>
				</div>
				<div class="day dispo">
					<a href="#" data-date="2025-07-20" id="date20250720" onclick="return false;">
						<span class="date">07/20</span>
						<span class="place">1</span>
					</a>
				</div>
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
            err := parseRefugeContent(tt.html, &tt.refuge, time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC))

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
