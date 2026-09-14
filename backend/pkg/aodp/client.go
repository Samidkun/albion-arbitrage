package aodp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"albionflow/backend/pkg/store"
)

// Default AODP endpoints per region.
const (
	DefaultAsiaEndpoint     = "https://east.albion-online-data.com"
	DefaultAmericasEndpoint = "https://west.albion-online-data.com"
	DefaultEuropeEndpoint   = "https://europe.albion-online-data.com"
)

// PriceResponseItem matches the AODP JSON response item for prices.
type PriceResponseItem struct {
	ItemID           string `json:"item_id"`
	City             string `json:"city"`
	Quality          int    `json:"quality"`
	SellPriceMin     int    `json:"sell_price_min"`
	SellPriceMinDate string `json:"sell_price_min_date"`
	BuyPriceMax      int    `json:"buy_price_max"`
	BuyPriceMaxDate  string `json:"buy_price_max_date"`
}

// HistoryDataPoint holds a single historical volume point from AODP.
type HistoryDataPoint struct {
	ItemCount    int64  `json:"item_count"`
	SilverAmount int64  `json:"silver_amount"`
	Timestamp    string `json:"timestamp"`
}

// HistoryResponseItem matches the AODP JSON response item for trade history.
type HistoryResponseItem struct {
	ItemID   string             `json:"item_id"`
	Location string             `json:"location"`
	Quality  int                `json:"quality"`
	Data     []HistoryDataPoint `json:"data"`
}

// Client interacts with The Albion Online Data Project (AODP) API across regions.
type Client struct {
	httpClient *http.Client
	baseURLs   map[string]string
}

// ClientOption configures a Client instance.
type ClientOption func(*Client)

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// WithBaseURL overrides the base URL for a specific region.
func WithBaseURL(region, baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURLs[strings.ToLower(strings.TrimSpace(region))] = strings.TrimRight(baseURL, "/")
	}
}

// WithBaseURLs overrides base URLs for multiple regions.
func WithBaseURLs(urls map[string]string) ClientOption {
	return func(c *Client) {
		for r, u := range urls {
			c.baseURLs[strings.ToLower(strings.TrimSpace(r))] = strings.TrimRight(u, "/")
		}
	}
}

// NewClient creates a new AODP client configured with default endpoints.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURLs: map[string]string{
			"asia":     DefaultAsiaEndpoint,
			"americas": DefaultAmericasEndpoint,
			"europe":   DefaultEuropeEndpoint,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Client) getBaseURL(region string) (string, error) {
	norm := strings.ToLower(strings.TrimSpace(region))
	baseURL, ok := c.baseURLs[norm]
	if !ok || baseURL == "" {
		return "", fmt.Errorf("unsupported region: %q", region)
	}
	return baseURL, nil
}

// FetchPrices queries AODP for the latest prices of the given itemIDs and locations.
func (c *Client) FetchPrices(ctx context.Context, region string, itemIDs []string, locations []string) ([]PriceResponseItem, error) {
	baseURL, err := c.getBaseURL(region)
	if err != nil {
		return nil, err
	}

	if len(itemIDs) == 0 {
		return []PriceResponseItem{}, nil
	}

	reqURL := fmt.Sprintf("%s/api/v2/stats/prices/%s.json", baseURL, strings.Join(itemIDs, ","))
	if len(locations) > 0 {
		q := url.Values{}
		q.Set("locations", strings.Join(locations, ","))
		reqURL += "?" + q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create prices request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch prices: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("prices api returned status %d: %s", resp.StatusCode, string(body))
	}

	var items []PriceResponseItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode prices json: %w", err)
	}

	return items, nil
}

// Fetch24hVolumes queries AODP for 24h history metrics of the given itemIDs and locations.
func (c *Client) Fetch24hVolumes(ctx context.Context, region string, itemIDs []string, locations []string) ([]HistoryResponseItem, error) {
	baseURL, err := c.getBaseURL(region)
	if err != nil {
		return nil, err
	}

	if len(itemIDs) == 0 {
		return []HistoryResponseItem{}, nil
	}

	reqURL := fmt.Sprintf("%s/api/v2/stats/history/%s.json", baseURL, strings.Join(itemIDs, ","))
	q := url.Values{}
	if len(locations) > 0 {
		q.Set("locations", strings.Join(locations, ","))
	}
	q.Set("time-scale", "24")
	reqURL += "?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create volumes request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch 24h volumes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("volumes api returned status %d: %s", resp.StatusCode, string(body))
	}

	var items []HistoryResponseItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("decode volumes json: %w", err)
	}

	return items, nil
}

var timestampFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05Z07:00",
}

func parseDataTimestamp(s string) (time.Time, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	for _, format := range timestampFormats {
		if t, err := time.Parse(format, trimmed); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown timestamp format: %s", s)
}

// IngestToStore fetches prices and 24h volumes for the specified items and locations,
// filters out empty price slots, and upserts valid records into the provided Store.
func (c *Client) IngestToStore(ctx context.Context, s *store.Store, region string, itemIDs []string, locations []string) (int, int, error) {
	normRegion := strings.ToLower(strings.TrimSpace(region))
	if _, err := c.getBaseURL(normRegion); err != nil {
		return 0, 0, err
	}

	prices, err := c.FetchPrices(ctx, normRegion, itemIDs, locations)
	if err != nil {
		return 0, 0, fmt.Errorf("fetch prices: %w", err)
	}

	pricesCount := 0
	for _, p := range prices {
		// Skip empty market slots (both sell min and buy max are 0)
		if p.SellPriceMin == 0 && p.BuyPriceMax == 0 {
			continue
		}

		snapshot := &store.PriceSnapshot{
			Region:           normRegion,
			ItemID:           p.ItemID,
			City:             p.City,
			Quality:          p.Quality,
			SellPriceMin:     p.SellPriceMin,
			SellPriceMinDate: p.SellPriceMinDate,
			BuyPriceMax:      p.BuyPriceMax,
			BuyPriceMaxDate:  p.BuyPriceMaxDate,
		}

		if err := s.UpsertPriceSnapshot(ctx, snapshot); err != nil {
			return pricesCount, 0, fmt.Errorf("upsert price snapshot for %s (%s): %w", p.ItemID, p.City, err)
		}
		pricesCount++
	}

	volumes, err := c.Fetch24hVolumes(ctx, normRegion, itemIDs, locations)
	if err != nil {
		return pricesCount, 0, fmt.Errorf("fetch volumes: %w", err)
	}

	volumesCount := 0
	for _, h := range volumes {
		if len(h.Data) == 0 {
			continue
		}

		// Pick the latest data point
		latest := h.Data[len(h.Data)-1]
		latestTime, errLatest := parseDataTimestamp(latest.Timestamp)

		for _, dp := range h.Data {
			t, err := parseDataTimestamp(dp.Timestamp)
			if err == nil && (errLatest != nil || t.After(latestTime)) {
				latestTime = t
				latest = dp
				errLatest = nil
			}
		}

		vol := &store.MarketVolume{
			Region:       normRegion,
			ItemID:       h.ItemID,
			City:         h.Location,
			Quality:      h.Quality,
			ItemCount:    latest.ItemCount,
			SilverAmount: latest.SilverAmount,
			Timestamp:    latest.Timestamp,
		}

		if err := s.UpsertMarketVolume(ctx, vol); err != nil {
			return pricesCount, volumesCount, fmt.Errorf("upsert market volume for %s (%s): %w", h.ItemID, h.Location, err)
		}
		volumesCount++
	}

	return pricesCount, volumesCount, nil
}
