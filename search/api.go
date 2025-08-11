package search

import (
	"encoding/json"
	"log"
	"net/http"

	"breeze-search/core"
	"breeze-search/crawler"
	"breeze-search/domains"
)

// API represents the REST API server for the search engine.
type API struct {
	Searcher      *Searcher
	Crawler       *crawler.Crawler
	DomainManager *domains.DomainManager
}

// NewAPI creates a new API instance.
func NewAPI(searcher *Searcher, crawler *crawler.Crawler, domainManager *domains.DomainManager) *API {
	return &API{
		Searcher:      searcher,
		Crawler:       crawler,
		DomainManager: domainManager,
	}
}

// RegisterRoutes registers the API endpoints.
func (api *API) RegisterRoutes() {
	http.HandleFunc("/search", api.handleSearch)
	http.HandleFunc("/crawl", api.handleCrawl)
}

// handleSearch handles search requests.
func (api *API) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	results, err := api.Searcher.Search(query)
	if err != nil {
		http.Error(w, "Search failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// handleCrawl handles requests to start the crawler.
func (api *API) handleCrawl(w http.ResponseWriter, r *http.Request) {
	log.Println("Crawl requested via API.")

	go func() {
		api.Crawler.Start()
		defer api.Crawler.Stop()

		domainsToCrawl := api.DomainManager.GetAllDomains()
		log.Printf("Enqueuing %d domains for crawling...", len(domainsToCrawl))

		for _, d := range domainsToCrawl {
			api.Crawler.EnqueueJob(core.CrawlJob{URL: d.URL})
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Crawling started in the background."})
}