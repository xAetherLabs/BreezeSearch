package main

import (
	"log"
	"time"

	"breeze-search/core"
	"breeze-search/crawler"
	"breeze-search/domains"
	"breeze-search/indexer"
	"breeze-search/storage"
	"breeze-search/search"
)

func main() {
	// 1. Initialize components
	domainManager := domains.NewDomainManager("domains/solana_sites.json")
	if err := domainManager.LoadDomains(); err != nil {
		log.Fatalf("Failed to load domains: %v", err)
	}

	badgerStore, err := storage.NewBadgerStore("badger_data")
	if err != nil {
		log.Fatalf("Failed to create BadgerStore: %v", err)
	}
	defer badgerStore.Close()

	bleveIndex, err := indexer.NewBleveIndex("bleve_index")
	if err != nil {
		log.Fatalf("Failed to create BleveIndex: %v", err)
	}
	defer bleveIndex.Close()

	crawlerConfig := &crawler.CrawlerConfig{
		MaxDepth:           1,
		MaxConcurrentPages: 2,
		RequestTimeout:     10 * time.Second,
		UserAgent:          "Breeze-Search-Bot/1.0",
	}

	crawler := crawler.NewCrawler(crawlerConfig, domainManager, bleveIndex, badgerStore)
	crawler.Start()
	defer crawler.Stop()

	// 2. Enqueue some test domains
	testDomains := []string{
		"https://solana.com",
		"https://www.tensor.trade/",
		"https://www.magiceden.io/",
	}

	for _, u := range testDomains {
		domain, err := domainManager.GetDomainByURL(u)
		if err != nil {
			domain = &domains.Domain{Name: u, URL: u}
		}
		crawler.EnqueueJob(core.CrawlJob{URL: domain.URL})
	}

	time.Sleep(20 * time.Second) // Wait for crawls to complete

	searcher := search.NewSearcher(bleveIndex, badgerStore)

	// 3. Perform test searches
	log.Println("--- SEARCH RESULTS ---")
	searchQueries := []string{
		"solana",
		"tensor",
		"magiceden",
	}

	for _, query := range searchQueries {
		log.Printf("Searching for: %s", query)
		results, err := searcher.Search(query)
		if err != nil {
			log.Printf("Error searching: %v", err)
			continue
		}
		log.Printf("  Found %d results.", len(results))
		for _, doc := range results {
			log.Printf("    - %s", doc.URL)
		}
	}
}
