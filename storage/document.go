package storage

import "time"

// Document represents a single crawled and processed web page.
type Document struct {
	ID          string    `json:"id"`          // Unique identifier (e.g., hash of the URL)
	URL         string    `json:"url"`         // Original URL of the page
	Title       string    `json:"title"`       // Extracted title
	Description string    `json:"description"` // Extracted meta description
	BodyText    string    `json:"body_text"`    // Cleaned body text
	Links       []string  `json:"links"`       // Outgoing links
	ContentHash string    `json:"content_hash"` // Hash of the raw content to detect changes
	CrawledAt   time.Time `json:"crawled_at"`   // When the page was last crawled
	IndexedAt   time.Time `json:"indexed_at"`   // When the page was last indexed
}
