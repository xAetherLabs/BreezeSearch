package indexer

import (
	"log"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/keyword"

	"breeze-search/storage"
)

// BleveIndex is a SearchIndex implementation using Bleve.
type BleveIndex struct {
	index bleve.Index
}

// NewBleveIndex creates a new BleveIndex.
func NewBleveIndex(path string) (*BleveIndex, error) {
	mapping := BuildMapping()
	index, err := bleve.New(path, mapping)
	if err != nil {
		index, err = bleve.Open(path)
		if err != nil {
			return nil, err
		}
	}
	return &BleveIndex{index: index}, nil
}

// Index indexes a document.
func (b *BleveIndex) Index(doc *storage.Document) error {
	log.Printf("Indexing document: ID=%s, URL=%s, Title=%s, Description=%s, BodyTextLength=%d", doc.ID, doc.URL, doc.Title, doc.Description, len(doc.BodyText))
	return b.index.Index(doc.ID, doc)
}

// Search performs a search query.
func (b *BleveIndex) Search(query string) ([]string, error) {
	searchRequest := bleve.NewSearchRequest(bleve.NewQueryStringQuery(query))
	searchResult, err := b.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	var docIDs []string
	for _, hit := range searchResult.Hits {
		docIDs = append(docIDs, hit.ID)
	}
	return docIDs, nil
}

// Close closes the index.
func (b *BleveIndex) Close() error {
	return b.index.Close()
}

// BuildMapping creates a Bleve index mapping for the Document struct.
func BuildMapping() *mapping.IndexMappingImpl {
	mappingImpl := bleve.NewIndexMapping()

	// Document mapping
	docMapping := bleve.NewDocumentMapping()

	// Title
	titleFieldMapping := bleve.NewTextFieldMapping()
	titleFieldMapping.Analyzer = standard.Name
	titleFieldMapping.Store = true
	titleFieldMapping.IncludeInAll = true
	docMapping.AddFieldMappingsAt("Title", titleFieldMapping)

	// Description
	descFieldMapping := bleve.NewTextFieldMapping()
	descFieldMapping.Analyzer = standard.Name
	descFieldMapping.Store = true
	descFieldMapping.IncludeInAll = true
	docMapping.AddFieldMappingsAt("Description", descFieldMapping)

	// Body
	bodyFieldMapping := bleve.NewTextFieldMapping()
	bodyFieldMapping.Analyzer = standard.Name
	bodyFieldMapping.Store = true
	bodyFieldMapping.IncludeInAll = true
	docMapping.AddFieldMappingsAt("BodyText", bodyFieldMapping)

	// URL (keyword analyzer)
	urlFieldMapping := bleve.NewTextFieldMapping()
	urlFieldMapping.Analyzer = keyword.Name
	docMapping.AddFieldMappingsAt("URL", urlFieldMapping)

	// ID (keyword analyzer)
	idFieldMapping := bleve.NewTextFieldMapping()
	idFieldMapping.Analyzer = keyword.Name
	docMapping.AddFieldMappingsAt("ID", idFieldMapping)

	mappingImpl.AddDocumentMapping("document", docMapping)

	return mappingImpl
}
