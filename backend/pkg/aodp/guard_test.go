package aodp_test

import (
	"errors"
	"testing"
	"time"

	"albionflow/backend/pkg/aodp"
	"albionflow/backend/pkg/store"
)

func TestClassifyFreshness(t *testing.T) {
	tests := []struct {
		name     string
		age      time.Duration
		expected aodp.FreshnessTier
	}{
		{name: "10 minutes", age: 10 * time.Minute, expected: aodp.FreshnessFresh},
		{name: "29m59s", age: 29*time.Minute + 59*time.Second, expected: aodp.FreshnessFresh},
		{name: "30m00s", age: 30 * time.Minute, expected: aodp.FreshnessAging},
		{name: "45 minutes", age: 45 * time.Minute, expected: aodp.FreshnessAging},
		{name: "60m00s", age: 60 * time.Minute, expected: aodp.FreshnessAging},
		{name: "60m01s", age: 60*time.Minute + 1*time.Second, expected: aodp.FreshnessStale},
		{name: "24 hours", age: 24 * time.Hour, expected: aodp.FreshnessStale},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := aodp.ClassifyFreshness(tc.age)
			if got != tc.expected {
				t.Fatalf("ClassifyFreshness(%v) = %v, expected %v", tc.age, got, tc.expected)
			}
		})
	}
}

func TestParseAODPDate(t *testing.T) {
	tests := []struct {
		name       string
		dateStr    string
		expectErr  bool
		isZeroDate bool
		expected   time.Time
	}{
		{
			name:      "simple ISO without timezone",
			dateStr:   "2026-09-14T10:15:30",
			expectErr: false,
			expected:  time.Date(2026, 9, 14, 10, 15, 30, 0, time.UTC),
		},
		{
			name:      "RFC3339 with Z",
			dateStr:   "2026-09-14T10:15:30Z",
			expectErr: false,
			expected:  time.Date(2026, 9, 14, 10, 15, 30, 0, time.UTC),
		},
		{
			name:      "RFC3339 with offset",
			dateStr:   "2026-09-14T12:15:30+02:00",
			expectErr: false,
			expected:  time.Date(2026, 9, 14, 10, 15, 30, 0, time.UTC),
		},
		{
			name:      "with milliseconds and Z",
			dateStr:   "2026-09-14T10:15:30.500Z",
			expectErr: false,
			expected:  time.Date(2026, 9, 14, 10, 15, 30, 500000000, time.UTC),
		},
		{
			name:       "empty date string",
			dateStr:    "",
			expectErr:  true,
			isZeroDate: true,
		},
		{
			name:       "zero date string 0001-01-01T00:00:00",
			dateStr:    "0001-01-01T00:00:00",
			expectErr:  true,
			isZeroDate: true,
		},
		{
			name:       "garbage string",
			dateStr:    "not-a-valid-date",
			expectErr:  true,
			isZeroDate: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := aodp.ParseAODPDate(tc.dateStr)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error for %q, got nil", tc.dateStr)
				}
				if tc.isZeroDate && !errors.Is(err, aodp.ErrZeroDate) {
					t.Fatalf("expected ErrZeroDate for %q, got: %v", tc.dateStr, err)
				}
				if !got.IsZero() {
					t.Fatalf("expected zero time on error, got %v", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.dateStr, err)
			}
			if !got.Equal(tc.expected) {
				t.Fatalf("ParseAODPDate(%q) = %v, expected %v", tc.dateStr, got, tc.expected)
			}
		})
	}
}

func TestCalculatePriceAge(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

	t.Run("valid date 15m ago", func(t *testing.T) {
		dateStr := "2026-09-14T11:45:00Z"
		recordedAt := now.Add(-10 * time.Minute)
		age, tier := aodp.CalculatePriceAge(dateStr, recordedAt, now)
		if age != 15*time.Minute {
			t.Fatalf("expected age 15m, got %v", age)
		}
		if tier != aodp.FreshnessFresh {
			t.Fatalf("expected tier %v, got %v", aodp.FreshnessFresh, tier)
		}
	})

	t.Run("fallback to recordedAt when date is zero", func(t *testing.T) {
		dateStr := "0001-01-01T00:00:00"
		recordedAt := now.Add(-40 * time.Minute)
		age, tier := aodp.CalculatePriceAge(dateStr, recordedAt, now)
		if age != 40*time.Minute {
			t.Fatalf("expected age 40m, got %v", age)
		}
		if tier != aodp.FreshnessAging {
			t.Fatalf("expected tier %v, got %v", aodp.FreshnessAging, tier)
		}
	})

	t.Run("fallback to recordedAt when date is invalid", func(t *testing.T) {
		dateStr := "invalid-date"
		recordedAt := now.Add(-70 * time.Minute)
		age, tier := aodp.CalculatePriceAge(dateStr, recordedAt, now)
		if age != 70*time.Minute {
			t.Fatalf("expected age 70m, got %v", age)
		}
		if tier != aodp.FreshnessStale {
			t.Fatalf("expected tier %v, got %v", aodp.FreshnessStale, tier)
		}
	})

	t.Run("both dates zero returns 24h stale", func(t *testing.T) {
		age, tier := aodp.CalculatePriceAge("", time.Time{}, now)
		if age != 24*time.Hour {
			t.Fatalf("expected age 24h, got %v", age)
		}
		if tier != aodp.FreshnessStale {
			t.Fatalf("expected tier %v, got %v", aodp.FreshnessStale, tier)
		}
	})

	t.Run("future date clamped to 0", func(t *testing.T) {
		futureDate := "2026-09-14T12:05:00Z"
		age, tier := aodp.CalculatePriceAge(futureDate, time.Time{}, now)
		if age != 0 {
			t.Fatalf("expected age 0 for future date, got %v", age)
		}
		if tier != aodp.FreshnessFresh {
			t.Fatalf("expected tier %v, got %v", aodp.FreshnessFresh, tier)
		}
	})
}

func TestFilterFreshPrices(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

	p10m := &store.PriceSnapshot{
		ItemID:           "T4_BAG",
		SellPriceMinDate: now.Add(-10 * time.Minute).Format(time.RFC3339),
		RecordedAt:       now.Add(-10 * time.Minute),
	}
	p45m := &store.PriceSnapshot{
		ItemID:           "T5_BAG",
		SellPriceMinDate: now.Add(-45 * time.Minute).Format(time.RFC3339),
		RecordedAt:       now.Add(-45 * time.Minute),
	}
	p90m := &store.PriceSnapshot{
		ItemID:           "T6_BAG",
		SellPriceMinDate: now.Add(-90 * time.Minute).Format(time.RFC3339),
		RecordedAt:       now.Add(-90 * time.Minute),
	}

	prices := []*store.PriceSnapshot{p10m, p45m, p90m}

	t.Run("default maxAge 60m", func(t *testing.T) {
		fresh, stale := aodp.FilterFreshPrices(prices, now, 0)
		if len(fresh) != 2 {
			t.Fatalf("expected 2 fresh prices, got %d", len(fresh))
		}
		if len(stale) != 1 {
			t.Fatalf("expected 1 stale price, got %d", len(stale))
		}
		if fresh[0].ItemID != "T4_BAG" || fresh[1].ItemID != "T5_BAG" {
			t.Fatalf("unexpected fresh items: %+v", fresh)
		}
		if stale[0].ItemID != "T6_BAG" {
			t.Fatalf("unexpected stale items: %+v", stale)
		}
	})

	t.Run("custom maxAge 30m", func(t *testing.T) {
		fresh, stale := aodp.FilterFreshPrices(prices, now, 30*time.Minute)
		if len(fresh) != 1 {
			t.Fatalf("expected 1 fresh price, got %d", len(fresh))
		}
		if len(stale) != 2 {
			t.Fatalf("expected 2 stale prices, got %d", len(stale))
		}
		if fresh[0].ItemID != "T4_BAG" {
			t.Fatalf("unexpected fresh item: %+v", fresh)
		}
	})
}
