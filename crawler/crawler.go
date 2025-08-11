package crawler

import (
	"crypto/sha256"
	"fmt"
	"log"
	"sync"
	"time"

	"breeze-search/core"
	"breeze-search/domains"
	"breeze-search/indexer"
	"breeze-search/storage"
)

// CrawlerConfig holds configuration settings for the crawler.
type CrawlerConfig struct {
	MaxDepth           int
	MaxConcurrentPages int
	RequestTimeout     time.Duration
	UserAgent          string
}

// Crawler orchestrates the crawling process.
type Crawler struct {
	config        *CrawlerConfig
	domainManager *domains.DomainManager
	indexer       indexer.SearchIndex
	storage       storage.ContentStore
	jobQueue      chan core.CrawlJob
	resultChan    chan core.CrawlResult
	workers       []*Worker
	quit          chan bool
	wg            sync.WaitGroup // For tracking results processing
	workerWg      sync.WaitGroup // For tracking worker goroutines
	extractor     *Extractor
	robotsHandler *RobotsTxtHandler
}

// NewCrawler creates a new Crawler instance.
func NewCrawler(config *CrawlerConfig, dm *domains.DomainManager, idx indexer.SearchIndex, store storage.ContentStore) *Crawler {
	return &Crawler{
		config:        config,
		domainManager: dm,
		indexer:       idx,
		storage:       store,
		jobQueue:      make(chan core.CrawlJob, config.MaxConcurrentPages),
		resultChan:    make(chan core.CrawlResult, config.MaxConcurrentPages),
		quit:          make(chan bool),
		extractor:     NewExtractor(),
		robotsHandler: NewRobotsTxtHandler(),
	}
}

// Start begins the crawling process.
func (c *Crawler) Start() {
	log.Println("Crawler starting...")

	for i := 0; i < c.config.MaxConcurrentPages; i++ {
		worker := NewWorker(i+1, c.config, c.jobQueue, c.resultChan, c.extractor, c.robotsHandler, &c.workerWg)
		c.workers = append(c.workers, worker)
		worker.Start()
	}

	go c.processResults()

	log.Printf("Crawler started with %d workers.", len(c.workers))
}

// Stop gracefully stops the crawling process.
func (c *Crawler) Stop() {
	log.Println("Crawler stopping...")
	close(c.quit)

	close(c.jobQueue)
	for _, worker := range c.workers {
		worker.Stop()
	}

	c.workerWg.Wait()
	close(c.resultChan)
	c.wg.Wait()

	log.Println("Crawler stopped.")
}

// EnqueueJob adds a new crawl job to the queue.
func (c *Crawler) EnqueueJob(job core.CrawlJob) {
	c.wg.Add(1)
	c.jobQueue <- job
}

// processResults handles the results coming from workers.
func (c *Crawler) processResults() {
	for {
		select {
		case result := <-c.resultChan:
			c.handleCrawlResult(result)
			c.wg.Done()
		case <-c.quit:
			log.Println("Result processor stopping.")
			return
		}
	}
}

// handleCrawlResult processes a single crawl result.
func (c *Crawler) handleCrawlResult(result core.CrawlResult) {
	if result.Success {
		log.Printf("Successfully crawled: %s", result.URL)

		// Create a new document
		hash := sha256.Sum256([]byte(result.URL))
		doc := &storage.Document{
			ID:          fmt.Sprintf("%x", hash),
			URL:         result.URL,
			Title:       result.Title,
			Description: result.Description,
			BodyText:    result.BodyText,
			Links:       result.Links,
			ContentHash: result.ContentHash,
			CrawledAt:   result.LastCrawled,
			IndexedAt:   time.Now(),
		}

		// Store the document
		if err := c.storage.Store(doc); err != nil {
			log.Printf("Error storing document %s: %v", doc.URL, err)
			return
		}

		// Index the document
		log.Printf("Indexing document: ID=%s, URL=%s, Title=%s, Description=%s, BodyTextLength=%d", doc.ID, doc.URL, doc.Title, doc.Description, len(doc.BodyText))
		if err := c.indexer.Index(doc); err != nil {
			log.Printf("Error indexing document %s: %v", doc.URL, err)
			return
		}

		log.Printf("Indexed document from: %s", result.URL)

	} else {
		log.Printf("Failed to crawl %s: %v", result.URL, result.Error)
	}
}