package storage

import (
	"encoding/json"
	"log"

	"github.com/dgraph-io/badger/v3"
)

// BadgerStore is a ContentStore implementation using BadgerDB.
type BadgerStore struct {
	db *badger.DB
}

// NewBadgerStore creates a new BadgerStore.
func NewBadgerStore(path string) (*BadgerStore, error) {
	opts := badger.DefaultOptions(path)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &BadgerStore{db: db}, nil
}

// Store saves a new document to the database.
func (s *BadgerStore) Store(doc *Document) error {
	log.Printf("Storing document with ID: %s, URL: %s", doc.ID, doc.URL)
	return s.db.Update(func(txn *badger.Txn) error {
		data, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		return txn.Set([]byte(doc.ID), data)
	})
}

// Get retrieves a document by its ID.
func (s *BadgerStore) Get(id string) (*Document, error) {
	var doc Document
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(id))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &doc)
		})
	})
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// Update updates an existing document.
func (s *BadgerStore) Update(doc *Document) error {
	return s.Store(doc) // For Badger, update is the same as store
}

// Delete removes a document from the database.
func (s *BadgerStore) Delete(id string) error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(id))
	})
}

// Close closes the database connection.
func (s *BadgerStore) Close() error {
	return s.db.Close()
}
