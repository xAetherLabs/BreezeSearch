package indexer

import "breeze-search/storage"

// SearchIndex defines the interface for indexing and searching documents.
type SearchIndex interface {
	Index(doc *storage.Document) error
	Search(query string) ([]string, error)
	Close() error
}
