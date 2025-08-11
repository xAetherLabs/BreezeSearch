package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"breeze-search/crawler"
	"breeze-search/domains"
	"breeze-search/indexer"
	"breeze-search/search"
	"breeze-search/storage"
)

func main() {
	// 1. Initialize components
	log.Println("Initializing components...")
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

	searcher := search.NewSearcher(bleveIndex, badgerStore)

	// 2. Create a crawler instance (but don't start it yet)
	crawlerConfig := &crawler.CrawlerConfig{
		MaxDepth:           1,
		MaxConcurrentPages: 2,
		RequestTimeout:     30 * time.Second,
		UserAgent:          "Breeze-Search-Bot/1.0",
	}
	crawler := crawler.NewCrawler(crawlerConfig, domainManager, bleveIndex, badgerStore)

	// 3. Initialize and start the API server
	api := search.NewAPI(searcher, crawler, domainManager)
	port := os.Getenv("PORT")
	if port == "" {
    		port = "8080" // fallback for local runs
	}
	server := &http.Server{Addr: ":" + port}


	go func() {
		log.Println("Starting server on :8080")
		api.RegisterRoutes()
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start server: %v", err)
		}
	}()

	// 4. Wait for a shutdown signal
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Server stopped gracefully.")
}
