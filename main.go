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

	// 3. Determine port (Render provides PORT via env var)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // local default
	}
	server := &http.Server{
		Addr: ":" + port,
	}

	// 4. Start API server
	go func() {
		log.Printf("Starting server on :%s\n", port)
		api := search.NewAPI(searcher, crawler, domainManager)
		api.RegisterRoutes()

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start server: %v", err)
		}
	}()

	// 5. Graceful shutdown on interrupt/terminate
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan

	log.Println("Shutting down server...")

	// Allow up to 10s for cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shut down HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown failed: %v", err)
	}

	log.Println("Server stopped gracefully.")
}
