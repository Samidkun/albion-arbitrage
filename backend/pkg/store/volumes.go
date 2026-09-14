package store

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MarketVolume stores 24h trading volume metrics for an item in a specific city/region.
type MarketVolume struct {
	ID           int64     `json:"id"`
	Region       string    `json:"region"`
	ItemID       string    `json:"item_id"`
	City         string    `json:"city"`
	Quality      int       `json:"quality"`
	ItemCount    int64     `json:"item_count"`
	SilverAmount int64     `json:"silver_amount"`
	Timestamp    string    `json:"timestamp"`
	RecordedAt   time.Time `json:"recorded_at"`
}

// IsIlliquid checks if the trade volume falls below the minimum required liquidity threshold.
// Defaults minCount to 5 when minCount <= 0.
func (v *MarketVolume) IsIlliquid(minCount int64) bool {
	if minCount <= 0 {
		minCount = 5
	}
	if v == nil {
		return true
	}
	return v.ItemCount < minCount
}

// UpsertMarketVolume inserts or updates 24h market volume statistics.
func (s *Store) UpsertMarketVolume(ctx context.Context, v *MarketVolume) error {
	if v.Quality <= 0 {
		v.Quality = 1
	}
	if v.Timestamp == "" {
		v.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	query := `
	INSERT INTO market_volumes (
		region, item_id, city, quality,
		item_count, silver_amount, timestamp, recorded_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(region, item_id, city, quality) DO UPDATE SET
		item_count = excluded.item_count,
		silver_amount = excluded.silver_amount,
		timestamp = excluded.timestamp,
		recorded_at = CURRENT_TIMESTAMP
	RETURNING id, recorded_at;
	`

	var recordedRaw any
	err := s.db.QueryRowContext(
		ctx,
		query,
		v.Region,
		v.ItemID,
		v.City,
		v.Quality,
		v.ItemCount,
		v.SilverAmount,
		v.Timestamp,
	).Scan(&v.ID, &recordedRaw)
	if err != nil {
		return fmt.Errorf("upsert market volume: %w", err)
	}

	recTime, err := parseRecordedAt(recordedRaw)
	if err == nil {
		v.RecordedAt = recTime
	}

	return nil
}

// GetMarketVolume fetches 24h volume for a specific region, item, city, and quality.
func (s *Store) GetMarketVolume(ctx context.Context, region, itemID, city string, quality int) (*MarketVolume, error) {
	if quality <= 0 {
		quality = 1
	}

	query := `
	SELECT id, region, item_id, city, quality,
	       item_count, silver_amount, timestamp, recorded_at
	FROM market_volumes
	WHERE region = ? AND item_id = ? AND city = ? AND quality = ?
	LIMIT 1;
	`

	var v MarketVolume
	var recordedRaw any

	err := s.db.QueryRowContext(ctx, query, region, itemID, city, quality).Scan(
		&v.ID,
		&v.Region,
		&v.ItemID,
		&v.City,
		&v.Quality,
		&v.ItemCount,
		&v.SilverAmount,
		&v.Timestamp,
		&recordedRaw,
	)
	if err != nil {
		return nil, err
	}

	v.RecordedAt, _ = parseRecordedAt(recordedRaw)
	return &v, nil
}

// GetVolumesForItems returns volume data for a set of items in a region.
func (s *Store) GetVolumesForItems(ctx context.Context, region string, itemIDs []string) ([]*MarketVolume, error) {
	if len(itemIDs) == 0 {
		return []*MarketVolume{}, nil
	}

	query := strings.Builder{}
	query.WriteString(`
	SELECT id, region, item_id, city, quality,
	       item_count, silver_amount, timestamp, recorded_at
	FROM market_volumes
	WHERE region = ?
	`)

	args := []any{region}
	query.WriteString(" AND item_id IN (")
	for i, id := range itemIDs {
		if i > 0 {
			query.WriteString(",")
		}
		query.WriteString("?")
		args = append(args, id)
	}
	query.WriteString(")")
	query.WriteString(" ORDER BY item_id ASC, city ASC, quality ASC;")

	rows, err := s.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("query volumes for items: %w", err)
	}
	defer rows.Close()

	var results []*MarketVolume
	for rows.Next() {
		var v MarketVolume
		var recordedRaw any

		if err := rows.Scan(
			&v.ID,
			&v.Region,
			&v.ItemID,
			&v.City,
			&v.Quality,
			&v.ItemCount,
			&v.SilverAmount,
			&v.Timestamp,
			&recordedRaw,
		); err != nil {
			return nil, fmt.Errorf("scan market volume: %w", err)
		}

		v.RecordedAt, _ = parseRecordedAt(recordedRaw)
		results = append(results, &v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return results, nil
}
