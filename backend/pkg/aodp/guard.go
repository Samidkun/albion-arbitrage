package aodp

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"albionflow/backend/pkg/store"
)

// FreshnessTier classifies data staleness for market quotes.
type FreshnessTier string

const (
	FreshnessFresh FreshnessTier = "fresh" // < 30 minutes
	FreshnessAging FreshnessTier = "aging" // 30 to 60 minutes
	FreshnessStale FreshnessTier = "stale" // > 60 minutes
)

// ErrZeroDate indicates an uninitialized or empty timestamp string.
var ErrZeroDate = errors.New("zero date")

var aodpDateFormats = []string{
	"2006-01-02T15:04:05.000Z",
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05",
}

// ClassifyFreshness categorizes the age of market data into freshness tiers.
func ClassifyFreshness(age time.Duration) FreshnessTier {
	if age < 30*time.Minute {
		return FreshnessFresh
	}
	if age <= 60*time.Minute {
		return FreshnessAging
	}
	return FreshnessStale
}

// ParseAODPDate parses timestamp strings returned by AODP.
func ParseAODPDate(dateStr string) (time.Time, error) {
	trimmed := strings.TrimSpace(dateStr)
	if trimmed == "" || strings.HasPrefix(trimmed, "0001-01-01") {
		return time.Time{}, ErrZeroDate
	}

	for _, layout := range aodpDateFormats {
		if t, err := time.Parse(layout, trimmed); err == nil {
			if t.IsZero() || t.Year() <= 1 {
				return time.Time{}, ErrZeroDate
			}
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse AODP date: %s", dateStr)
}

// CalculatePriceAge computes the duration since quote observation and its freshness tier.
func CalculatePriceAge(sellPriceMinDate string, recordedAt time.Time, now time.Time) (time.Duration, FreshnessTier) {
	t, err := ParseAODPDate(sellPriceMinDate)
	if err == nil && !t.IsZero() {
		age := now.Sub(t)
		if age < 0 {
			age = 0
		}
		return age, ClassifyFreshness(age)
	}

	if recordedAt.IsZero() {
		return 24 * time.Hour, FreshnessStale
	}

	age := now.Sub(recordedAt)
	if age < 0 {
		age = 0
	}
	return age, ClassifyFreshness(age)
}

// FilterFreshPrices splits snapshots into fresh and stale subsets according to maxAge.
func FilterFreshPrices(prices []*store.PriceSnapshot, now time.Time, maxAge time.Duration) (fresh []*store.PriceSnapshot, stale []*store.PriceSnapshot) {
	if maxAge <= 0 {
		maxAge = 60 * time.Minute
	}

	for _, p := range prices {
		if p == nil {
			continue
		}
		age, _ := CalculatePriceAge(p.SellPriceMinDate, p.RecordedAt, now)
		if age <= maxAge {
			fresh = append(fresh, p)
		} else {
			stale = append(stale, p)
		}
	}
	return fresh, stale
}
