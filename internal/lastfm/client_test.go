package lastfm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// MockLimiter implements RateLimiter without actual rate limiting
type MockLimiter struct {
	callCount int
	retries   int
}

func (ml *MockLimiter) Wait(ctx context.Context) error {
	ml.callCount++
	return nil
}

func (ml *MockLimiter) DoWithRetry(ctx context.Context, fn func() error) error {
	return fn()
}

func TestClientFetchPage(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		q := r.URL.Query()
		if q.Get("method") != "user.getRecentTracks" {
			t.Errorf("method = %q, want user.getRecentTracks", q.Get("method"))
		}
		if q.Get("user") != "alice" {
			t.Errorf("user = %q, want alice", q.Get("user"))
		}

		// Return mock response with FlexText (object format)
		resp := RecentTracksResponse{}
		resp.RecentTracks.Track = []Track{
			{
				Name: FlexText{Text: "Track 1", MBID: "mbid1"},
				Date: struct {
					UTS  string `json:"uts"`
					Text string `json:"#text"`
				}{UTS: "1000", Text: "Oct 30, 2025"},
			},
			{
				Name: FlexText{Text: "Track 2", MBID: "mbid2"},
				Date: struct {
					UTS  string `json:"uts"`
					Text string `json:"#text"`
				}{UTS: "1001", Text: "Oct 30, 2025"},
			},
		}
		resp.RecentTracks.Attr.Page = "1"
		resp.RecentTracks.Attr.PerPage = "200"
		resp.RecentTracks.Attr.Total = "5000"
		resp.RecentTracks.Attr.TotalPages = "25"

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client
	limiter := &MockLimiter{}
	client := NewClient("test-api-key", limiter)
	client.BaseURL = server.URL + "/"

	// Fetch page
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	page, err := client.FetchPage(ctx, "alice", 0, 0, 1, 200)

	// Verify results
	if err != nil {
		t.Fatalf("FetchPage failed: %v", err)
	}
	if page == nil {
		t.Fatal("page should not be nil")
	}
	if len(page.Tracks) != 2 {
		t.Errorf("got %d tracks, want 2", len(page.Tracks))
	}
	if page.Page != 1 {
		t.Errorf("page = %d, want 1", page.Page)
	}
	if page.Total != 5000 {
		t.Errorf("total = %d, want 5000", page.Total)
	}
	if page.TotalPages != 25 {
		t.Errorf("total_pages = %d, want 25", page.TotalPages)
	}
	if page.Tracks[0].Name.Text != "Track 1" {
		t.Errorf("track[0].Name.Text = %q, want Track 1", page.Tracks[0].Name.Text)
	}
}

func TestClientFetchPageWithTimestamps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		// Verify timestamp parameters
		if q.Get("from") != "1000" {
			t.Errorf("from = %q, want 1000", q.Get("from"))
		}
		if q.Get("to") != "2000" {
			t.Errorf("to = %q, want 2000", q.Get("to"))
		}

		resp := RecentTracksResponse{}
		resp.RecentTracks.Track = []Track{}
		resp.RecentTracks.Attr.Page = "1"
		resp.RecentTracks.Attr.PerPage = "200"
		resp.RecentTracks.Attr.Total = "100"
		resp.RecentTracks.Attr.TotalPages = "1"

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	limiter := &MockLimiter{}
	client := NewClient("test-api-key", limiter)
	client.BaseURL = server.URL + "/"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	page, err := client.FetchPage(ctx, "alice", 1000, 2000, 1, 200)
	if err != nil {
		t.Fatalf("FetchPage failed: %v", err)
	}
	if page == nil {
		t.Fatal("page should not be nil")
	}
}

func TestClientFetchPageRateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer server.Close()

	limiter := &MockLimiter{}
	client := NewClient("test-api-key", limiter)
	client.BaseURL = server.URL + "/"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.FetchPage(ctx, "alice", 0, 0, 1, 200)
	if err == nil {
		t.Error("expected error for 429 response")
	}
}

func TestClientFetchPageServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	limiter := &MockLimiter{}
	client := NewClient("test-api-key", limiter)
	client.BaseURL = server.URL + "/"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.FetchPage(ctx, "alice", 0, 0, 1, 200)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestClientFetchPageAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := RecentTracksResponse{
			Error:   6,
			Message: "user not found",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	limiter := &MockLimiter{}
	client := NewClient("test-api-key", limiter)
	client.BaseURL = server.URL + "/"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.FetchPage(ctx, "nonexistent", 0, 0, 1, 200)
	if err == nil {
		t.Error("expected error for api error response")
	}
	if err.Error() != "api error 6: user not found" {
		t.Errorf("error = %q, want 'api error 6: user not found'", err)
	}
}

func TestIsNowPlaying(t *testing.T) {
	tests := []struct {
		name     string
		track    *Track
		expected bool
	}{
		{
			name: "now playing - has nowplaying field",
			track: &Track{
				NowPlaying: "true",
				Date: struct {
					UTS  string `json:"uts"`
					Text string `json:"#text"`
				}{UTS: "1000"},
			},
			expected: true,
		},
		{
			name: "now playing - no uts",
			track: &Track{
				NowPlaying: "",
				Date: struct {
					UTS  string `json:"uts"`
					Text string `json:"#text"`
				}{UTS: ""},
			},
			expected: true,
		},
		{
			name: "not now playing - has uts",
			track: &Track{
				NowPlaying: "",
				Date: struct {
					UTS  string `json:"uts"`
					Text string `json:"#text"`
				}{UTS: "1000"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNowPlaying(tt.track)
			if result != tt.expected {
				t.Errorf("IsNowPlaying = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClientContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		resp := RecentTracksResponse{}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	limiter := &MockLimiter{}
	client := NewClient("test-api-key", limiter)
	client.BaseURL = server.URL + "/"

	// Create context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.FetchPage(ctx, "alice", 0, 0, 1, 200)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestPaginationValidation(t *testing.T) {
	tests := []struct {
		pageSize int
		valid    bool
	}{
		{1, true},
		{50, true},
		{200, true},
		{0, false},
		{-1, false},
		{201, false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("pageSize=%d", tt.pageSize), func(t *testing.T) {
			err := ValidatePageSize(tt.pageSize)
			if (err == nil) != tt.valid {
				t.Errorf("ValidatePageSize(%d) valid=%v, want %v", tt.pageSize, err == nil, tt.valid)
			}
		})
	}
}

func TestFlexTextUnmarshalString(t *testing.T) {
	// Test unmarshaling plain string format
	tests := []struct {
		name     string
		json     string
		wantText string
		wantMBID string
		wantErr  bool
	}{
		{
			name:     "plain string",
			json:     `"My Track Name"`,
			wantText: "My Track Name",
			wantMBID: "",
			wantErr:  false,
		},
		{
			name:     "object with #text only",
			json:     `{"#text": "Track from Object"}`,
			wantText: "Track from Object",
			wantMBID: "",
			wantErr:  false,
		},
		{
			name:     "object with #text and mbid",
			json:     `{"#text": "Track with ID", "mbid": "abc-123"}`,
			wantText: "Track with ID",
			wantMBID: "abc-123",
			wantErr:  false,
		},
		{
			name:     "empty string",
			json:     `""`,
			wantText: "",
			wantMBID: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ft FlexText
			err := json.Unmarshal([]byte(tt.json), &ft)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if ft.Text != tt.wantText {
				t.Errorf("Text = %q, want %q", ft.Text, tt.wantText)
			}
			if ft.MBID != tt.wantMBID {
				t.Errorf("MBID = %q, want %q", ft.MBID, tt.wantMBID)
			}
		})
	}
}

func TestFlexTextRoundTrip(t *testing.T) {
	// Test that we can unmarshal both formats and re-marshal them
	tests := []struct {
		name    string
		jsonStr string
	}{
		{
			name:    "string format",
			jsonStr: `"Track Name"`,
		},
		{
			name:    "object format",
			jsonStr: `{"#text": "Track Name", "mbid": "id-123"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ft FlexText
			if err := json.Unmarshal([]byte(tt.jsonStr), &ft); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			if ft.Text != "Track Name" {
				t.Errorf("Text = %q, want Track Name", ft.Text)
			}
		})
	}
}

func TestTrackWithFlexTextFields(t *testing.T) {
	// Test full Track parsing with mixed FlexText formats
	jsonData := `{
		"name": "Song Name",
		"artist": {"#text": "Artist Name", "mbid": "artist-id"},
		"album": "Album Name",
		"date": {"uts": "1234567890", "#text": "Oct 30, 2025"},
		"mbid": "track-mbid"
	}`

	var track Track
	err := json.Unmarshal([]byte(jsonData), &track)
	if err != nil {
		t.Fatalf("Failed to unmarshal track: %v", err)
	}

	if track.Name.Text != "Song Name" {
		t.Errorf("Name.Text = %q, want Song Name", track.Name.Text)
	}
	if track.Artist.Text != "Artist Name" {
		t.Errorf("Artist.Text = %q, want Artist Name", track.Artist.Text)
	}
	if track.Artist.MBID != "artist-id" {
		t.Errorf("Artist.MBID = %q, want artist-id", track.Artist.MBID)
	}
	if track.Album.Text != "Album Name" {
		t.Errorf("Album.Text = %q, want Album Name", track.Album.Text)
	}
}

func TestTrackWithFlexTextObjectFormat(t *testing.T) {
	// Test Track parsing with all FlexText in object format
	jsonData := `{
		"name": {"#text": "Song Name", "mbid": "name-mbid"},
		"artist": {"#text": "Artist Name", "mbid": "artist-id"},
		"album": {"#text": "Album Name", "mbid": "album-id"},
		"date": {"uts": "1234567890", "#text": "Oct 30, 2025"}
	}`

	var track Track
	err := json.Unmarshal([]byte(jsonData), &track)
	if err != nil {
		t.Fatalf("Failed to unmarshal track: %v", err)
	}

	if track.Name.Text != "Song Name" || track.Name.MBID != "name-mbid" {
		t.Errorf("Name parsing failed: %+v", track.Name)
	}
	if track.Artist.Text != "Artist Name" || track.Artist.MBID != "artist-id" {
		t.Errorf("Artist parsing failed: %+v", track.Artist)
	}
	if track.Album.Text != "Album Name" || track.Album.MBID != "album-id" {
		t.Errorf("Album parsing failed: %+v", track.Album)
	}
}

func TestPaginatorShortCircuit(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		resp := RecentTracksResponse{}

		// First call returns one track, second call returns empty
		if callCount == 1 {
			resp.RecentTracks.Track = []Track{
				{Name: FlexText{Text: "Track 1"}},
			}
		} else {
			resp.RecentTracks.Track = []Track{}
		}

		resp.RecentTracks.Attr.Page = fmt.Sprintf("%d", callCount)
		resp.RecentTracks.Attr.PerPage = "200"
		resp.RecentTracks.Attr.Total = "1"
		resp.RecentTracks.Attr.TotalPages = "1"

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	limiter := &MockLimiter{}
	client := NewClient("test-api-key", limiter)
	client.BaseURL = server.URL + "/"

	paginator := NewPaginator(client, "alice", 0, 0, 200, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First page should succeed
	page1, err := paginator.Next(ctx)
	if err != nil {
		t.Fatalf("Next() failed: %v", err)
	}
	if page1 == nil {
		t.Fatal("page1 should not be nil")
	}

	// Second page returns no records, should short-circuit
	page2, err := paginator.Next(ctx)
	if err != nil {
		t.Fatalf("Next() failed: %v", err)
	}
	if page2 != nil {
		t.Error("page2 should be nil (short-circuit on empty)")
	}
}
