package storage

// ContentStore defines the interface for storing and retrieving documents.
type ContentStore interface {
	Store(doc *Document) error
	Get(id string) (*Document, error)
	Update(doc *Document) error
	Delete(id string) error
	Close() error
}
