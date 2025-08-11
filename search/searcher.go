package search

import (
	"breeze-search/indexer"
	"breeze-search/storage"
	"log"
)

// Searcher provides search capabilities over indexed documents.
type Searcher struct {
	BleveIndex *indexer.BleveIndex
	BadgerStore *storage.BadgerStore
}

// NewSearcher creates a new Searcher instance.
func NewSearcher(bleveIdx *indexer.BleveIndex, badgerStore *storage.BadgerStore) *Searcher {
	return &Searcher{
		BleveIndex: bleveIdx,
		BadgerStore: badgerStore,
	}
}

// Search performs a search across indexed documents using Bleve.
func (s *Searcher) Search(query string) ([]*storage.Document, error) {
	hitIDs, err := s.BleveIndex.Search(query)
	if err != nil {
		return nil, err
	}

	var results []*storage.Document
	for _, docID := range hitIDs {
		doc, err := s.BadgerStore.Get(docID)
		if err != nil {
			// Log the error but continue with other documents
			log.Printf("Error retrieving document from BadgerStore: %v", err)
			continue
		}
		results = append(results, doc)
	}
	return results, nil
}
