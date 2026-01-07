package lastfm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/lastfm-reader/lastfm-sync/internal/ratelimit"
)

// FlexText unmarshals either a plain string or an object with #text and optional mbid
type FlexText struct {
	Text string
	MBID string
}

// UnmarshalJSON handles both string and object formats from Last.fm API
// Format 1: "track name" (plain string)
// Format 2: {"#text": "track name", "mbid": "id"} (object)
func (ft *FlexText) UnmarshalJSON(data []byte) error {
	// Try plain string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		ft.Text = s
		ft.MBID = ""
		return nil
	}

	// Try object with #text and mbid
	var obj struct {
		Text string `json:"#text"`
		MBID string `json:"mbid"`
	}
	if err := json.Unmarshal(data, &obj); err == nil {
		ft.Text = obj.Text
		ft.MBID = obj.MBID
		return nil
	}

	return fmt.Errorf("cannot unmarshal FlexText from %s", string(data))
}

// MarshalJSON exports FlexText back to JSON (as plain string for compatibility)
func (ft *FlexText) MarshalJSON() ([]byte, error) {
	return json.Marshal(ft.Text)
}

// Client handles Last.fm API requests with retry and rate limiting
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	Limiter    RateLimiter
}

// RateLimiter interface for pluggable rate limiting (mocked in tests)
type RateLimiter interface {
	Wait(ctx context.Context) error
	DoWithRetry(ctx context.Context, fn func() error) error
}

// NewClient creates a Last.fm API client with default settings
func NewClient(apiKey string, limiter RateLimiter) *Client {
	if limiter == nil {
		panic("limiter cannot be nil")
	}
	return &Client{
		BaseURL: "https://ws.audioscrobbler.com/2.0/",
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		Limiter: limiter,
	}
}

// Track represents a single track from Last.fm API
type Track struct {
	Artist FlexText `json:"artist"`
	Album  FlexText `json:"album"`
	Name   FlexText `json:"name"`
	Date   struct {
		UTS  string `json:"uts"`
		Text string `json:"#text"`
	} `json:"date"`
	NowPlaying string `json:"nowplaying,omitempty"`
	MBID       string `json:"mbid"`
	URL        string `json:"url"`
	Image      []struct {
		Size string `json:"size"`
		Text string `json:"#text"`
	} `json:"image"`
	Raw json.RawMessage `json:"-"` // Original JSON (set after unmarshaling)
}

// RecentTracksResponse is the Last.fm API response for user.getRecentTracks
type RecentTracksResponse struct {
	RecentTracks struct {
		Track []Track `json:"track"`
		Attr  struct {
			Page       string `json:"page"`
			PerPage    string `json:"perPage"`
			Total      string `json:"total"`
			TotalPages string `json:"totalPages"`
		} `json:"@attr"`
	} `json:"recenttracks"`
	Error   int    `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// Page represents a single page of results
type Page struct {
	Tracks     []Track
	Page       int
	PerPage    int
	Total      int
	TotalPages int
}

// FetchPage fetches a single page of recent tracks for a user
// from, to: unix timestamps (0 means no limit)
func (c *Client) FetchPage(ctx context.Context, username string, from, to int64, pageNum, pageSize int) (*Page, error) {
	var page *Page

	// Use rate limiter with retry
	retryErr := c.Limiter.DoWithRetry(ctx, func() error {
		var fetchErr error
		page, fetchErr = c.fetchPageDirect(ctx, username, from, to, pageNum, pageSize)
		return fetchErr
	})

	return page, retryErr
}

// fetchPageDirect makes the actual HTTP request without retry
func (c *Client) fetchPageDirect(ctx context.Context, username string, from, to int64, pageNum, pageSize int) (*Page, error) {
	// Build query parameters
	params := map[string]string{
		"method":  "user.getRecentTracks",
		"user":    username,
		"api_key": c.APIKey,
		"format":  "json",
		"limit":   strconv.Itoa(pageSize),
		"page":    strconv.Itoa(pageNum),
	}

	if from > 0 {
		params["from"] = strconv.FormatInt(from, 10)
	}
	if to > 0 {
		params["to"] = strconv.FormatInt(to, 10)
	}

	// Build URL with query string
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	for key, value := range params {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		// 429 Too Many Requests - transient, retry with Retry-After
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, &ratelimit.HTTPError{
				Err:      fmt.Errorf("rate limited (429)"),
				Response: resp,
			}
		}
		// 5xx - transient, retry with Retry-After
		if resp.StatusCode >= 500 && resp.StatusCode < 600 {
			return nil, &ratelimit.HTTPError{
				Err:      fmt.Errorf("server error (%d)", resp.StatusCode),
				Response: resp,
			}
		}
		// Other errors - not transient
		_ = resp.Body
		return nil, fmt.Errorf("http error %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var apiResp RecentTracksResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if apiResp.Error != 0 {
		return nil, fmt.Errorf("api error %d: %s", apiResp.Error, apiResp.Message)
	}

	// Parse pagination info
	page, _ := strconv.Atoi(apiResp.RecentTracks.Attr.Page)
	perPage, _ := strconv.Atoi(apiResp.RecentTracks.Attr.PerPage)
	total, _ := strconv.Atoi(apiResp.RecentTracks.Attr.Total)
	totalPages, _ := strconv.Atoi(apiResp.RecentTracks.Attr.TotalPages)

	// Handle parsing errors
	if page < 0 || perPage < 0 || total < 0 || totalPages < 0 {
		return nil, fmt.Errorf("invalid pagination info in response")
	}

	return &Page{
		Tracks:     apiResp.RecentTracks.Track,
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// IsNowPlaying checks if a track is the currently playing track (no date)
func IsNowPlaying(track *Track) bool {
	return track.NowPlaying != "" || track.Date.UTS == ""
}
