package store

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PriceSnapshot holds the latest market price data for an item in a specific city and region.
type PriceSnapshot struct {
	ID               int64     `json:"id"`
	Region           string    `json:"region"`
	ItemID           string    `json:"item_id"`
	City             string    `json:"city"`
	Quality          int       `json:"quality"`
	SellPriceMin     int       `json:"sell_price_min"`
	SellPriceMinDate string    `json:"sell_price_min_date"`
	BuyPriceMax      int       `json:"buy_price_max"`
	BuyPriceMaxDate  string    `json:"buy_price_max_date"`
	RecordedAt       time.Time `json:"recorded_at"`
}

var marketTimeFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z07:00",
}

// parseMarketTime parses a date string across supported AODP / SQLite formats.
func parseMarketTime(dateStr string) (time.Time, error) {
	trimmed := strings.TrimSpace(dateStr)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}

	for _, format := range marketTimeFormats {
		if t, err := time.Parse(format, trimmed); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time string: %s", dateStr)
}

// parseRecordedAt converts SQLite dynamic timestamp types into time.Time.
func parseRecordedAt(val any) (time.Time, error) {
	switch v := val.(type) {
	case time.Time:
		return v.UTC(), nil
	case string:
		return parseMarketTime(v)
	case []byte:
		return parseMarketTime(string(v))
	default:
		return time.Time{}, fmt.Errorf("unexpected time type: %T", val)
	}
}

// Age computes how old the market price is relative to the given reference time.
// It parses SellPriceMinDate and falls back to RecordedAt if empty, zero, or invalid.
func (p *PriceSnapshot) Age(now time.Time) time.Duration {
	refTime := p.RecordedAt

	if p.SellPriceMinDate != "" && !strings.HasPrefix(p.SellPriceMinDate, "0001-01-01") {
		if parsed, err := parseMarketTime(p.SellPriceMinDate); err == nil && parsed.Year() > 1 {
			refTime = parsed
		}
	}

	if refTime.IsZero() {
		return 0
	}

	age := now.Sub(refTime)
	if age < 0 {
		return 0
	}
	return age
}

// IsStale returns true if the price snapshot is older than threshold,
// or if SellPriceMinDate indicates a missing or zero record.
func (p *PriceSnapshot) IsStale(now time.Time, threshold time.Duration) bool {
	if p.SellPriceMinDate == "" || strings.HasPrefix(p.SellPriceMinDate, "0001-01-01") {
		return true
	}
	return p.Age(now) > threshold
}

// UpsertPriceSnapshot inserts or updates a price snapshot record.
func (s *Store) UpsertPriceSnapshot(ctx context.Context, p *PriceSnapshot) error {
	if p.Quality <= 0 {
		p.Quality = 1
	}
	if p.SellPriceMinDate == "" {
		p.SellPriceMinDate = "0001-01-01T00:00:00"
	}
	if p.BuyPriceMaxDate == "" {
		p.BuyPriceMaxDate = "0001-01-01T00:00:00"
	}

	query := `
	INSERT INTO price_snapshots (
		region, item_id, city, quality,
		sell_price_min, sell_price_min_date,
		buy_price_max, buy_price_max_date,
		recorded_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(region, item_id, city, quality) DO UPDATE SET
		sell_price_min = excluded.sell_price_min,
		sell_price_min_date = excluded.sell_price_min_date,
		buy_price_max = excluded.buy_price_max,
		buy_price_max_date = excluded.buy_price_max_date,
		recorded_at = CURRENT_TIMESTAMP
	RETURNING id, recorded_at;
	`

	var recordedRaw any
	err := s.db.QueryRowContext(
		ctx,
		query,
		p.Region,
		p.ItemID,
		p.City,
		p.Quality,
		p.SellPriceMin,
		p.SellPriceMinDate,
		p.BuyPriceMax,
		p.BuyPriceMaxDate,
	).Scan(&p.ID, &recordedRaw)
	if err != nil {
		return fmt.Errorf("upsert price snapshot: %w", err)
	}

	recTime, err := parseRecordedAt(recordedRaw)
	if err == nil {
		p.RecordedAt = recTime
	}

	return nil
}

// GetLatestPrice returns the price snapshot for a specific region, item, city, and quality.
func (s *Store) GetLatestPrice(ctx context.Context, region, itemID, city string, quality int) (*PriceSnapshot, error) {
	if quality <= 0 {
		quality = 1
	}

	query := `
	SELECT id, region, item_id, city, quality,
	       sell_price_min, sell_price_min_date,
	       buy_price_max, buy_price_max_date,
	       recorded_at
	FROM price_snapshots
	WHERE region = ? AND item_id = ? AND city = ? AND quality = ?
	LIMIT 1;
	`

	var p PriceSnapshot
	var recordedRaw any

	err := s.db.QueryRowContext(ctx, query, region, itemID, city, quality).Scan(
		&p.ID,
		&p.Region,
		&p.ItemID,
		&p.City,
		&p.Quality,
		&p.SellPriceMin,
		&p.SellPriceMinDate,
		&p.BuyPriceMax,
		&p.BuyPriceMaxDate,
		&recordedRaw,
	)
	if err != nil {
		return nil, err
	}

	p.RecordedAt, _ = parseRecordedAt(recordedRaw)
	return &p, nil
}

// GetPricesForItems returns price snapshots for a set of items in a region, optionally filtered by cities.
func (s *Store) GetPricesForItems(ctx context.Context, region string, itemIDs []string, cities []string) ([]*PriceSnapshot, error) {
	if len(itemIDs) == 0 {
		return []*PriceSnapshot{}, nil
	}

	query := strings.Builder{}
	query.WriteString(`
	SELECT id, region, item_id, city, quality,
	       sell_price_min, sell_price_min_date,
	       buy_price_max, buy_price_max_date,
	       recorded_at
	FROM price_snapshots
	WHERE region = ?
	`)

	args := []any{region}

	// Filter by itemIDs
	query.WriteString(" AND item_id IN (")
	for i, id := range itemIDs {
		if i > 0 {
			query.WriteString(",")
		}
		query.WriteString("?")
		args = append(args, id)
	}
	query.WriteString(")")

	// Filter by cities if provided
	if len(cities) > 0 {
		query.WriteString(" AND city IN (")
		for i, city := range cities {
			if i > 0 {
				query.WriteString(",")
			}
			query.WriteString("?")
			args = append(args, city)
		}
		query.WriteString(")")
	}

	query.WriteString(" ORDER BY item_id ASC, city ASC, quality ASC;")

	rows, err := s.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query prices for items: %w", err)
	}
	defer rows.Close()

	var results []*PriceSnapshot
	for rows.Next() {
		var p PriceSnapshot
		var recordedRaw any

		if err := rows.Scan(
			&p.ID,
			&p.Region,
			&p.ItemID,
			&p.City,
			&p.Quality,
			&p.SellPriceMin,
			&p.SellPriceMinDate,
			&p.BuyPriceMax,
			&p.BuyPriceMaxDate,
			&recordedRaw,
		); err != nil {
			return nil, fmt.Errorf("scan price snapshot: %w", err)
		}

		p.RecordedAt, _ = parseRecordedAt(recordedRaw)
		results = append(results, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return results, nil
}
