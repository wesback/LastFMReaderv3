package lastfm

import (
	"context"
	"fmt"
)

// Paginator iterates through pages of recent tracks
type Paginator struct {
	client   *Client
	username string
	from     int64
	to       int64
	pageSize int
	maxPages int
	current  int
}

// NewPaginator creates a paginator for a user's recent tracks
func NewPaginator(client *Client, username string, from, to int64, pageSize, maxPages int) *Paginator {
	return &Paginator{
		client:   client,
		username: username,
		from:     from,
		to:       to,
		pageSize: pageSize,
		maxPages: maxPages,
		current:  1,
	}
}

// Next retrieves the next page of tracks
// Returns nil when all pages exhausted or max-pages reached
func (p *Paginator) Next(ctx context.Context) (*Page, error) {
	// Check max-pages limit
	if p.maxPages > 0 && p.current > p.maxPages {
		return nil, nil
	}

	page, err := p.client.FetchPage(ctx, p.username, p.from, p.to, p.current, p.pageSize)
	if err != nil {
		return nil, err
	}

	// Short-circuit: if this page has no records, stop pagination
	if len(page.Tracks) == 0 {
		return nil, nil
	}

	// Move to next page for subsequent calls
	p.current++

	// Stop if we've fetched all pages
	if page.Page >= page.TotalPages {
		return page, nil
	}

	return page, nil
}

// PageIterator provides a functional iterator for pagination
type PageIterator struct {
	paginator *Paginator
	ctx       context.Context
	err       error
	current   *Page
}

// NewPageIterator creates a functional iterator
func NewPageIterator(ctx context.Context, paginator *Paginator) *PageIterator {
	return &PageIterator{
		paginator: paginator,
		ctx:       ctx,
	}
}

// HasNext checks if there are more pages
func (pi *PageIterator) HasNext() bool {
	if pi.err != nil {
		return false
	}

	var err error
	pi.current, err = pi.paginator.Next(pi.ctx)
	if err != nil {
		pi.err = err
		return false
	}

	return pi.current != nil
}

// Current returns the current page
func (pi *PageIterator) Current() *Page {
	return pi.current
}

// Error returns any error encountered during iteration
func (pi *PageIterator) Error() error {
	return pi.err
}

// PageNumber returns the current page number
func (p *Paginator) PageNumber() int {
	return p.current - 1 // current is incremented after fetch
}

// TotalPages returns the expected total pages (from first page response)
// Returns 0 before first page is fetched
func (p *Paginator) TotalPages() int {
	if p.current <= 1 {
		return 0
	}
	// This would need to be tracked differently in real usage
	// For now, return a placeholder
	return 0
}

// ValidatePageSize ensures page size is within Last.fm limits
func ValidatePageSize(pageSize int) error {
	if pageSize < 1 {
		return fmt.Errorf("page size must be at least 1, got %d", pageSize)
	}
	if pageSize > 200 {
		return fmt.Errorf("page size cannot exceed 200, got %d", pageSize)
	}
	return nil
}
