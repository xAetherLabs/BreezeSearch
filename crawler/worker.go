package crawler

import (
	"log"

	"breeze-search/core"
)

// Worker is a struct that represents a crawler worker.
type Worker struct {
	id         int
	jobQueue   <-chan core.CrawlJob
	resultChan chan<- core.CrawlResult
	quit       chan bool
	crawler    *Crawler // Reference to the main Crawler instance
}

// NewWorker creates a new Worker.
func NewWorker(id int, jobQueue <-chan core.CrawlJob, resultChan chan<- core.CrawlResult, crawler *Crawler) *Worker {
	return &Worker{
		id:         id,
		jobQueue:   jobQueue,
		resultChan: resultChan,
		quit:       make(chan bool),
		crawler:    crawler,
	}
}

// Start starts the worker's job processing loop.
func (w *Worker) Start() {
	w.crawler.workerWg.Add(1)
	go func() {
		defer w.crawler.workerWg.Done()
		for {
			select {
			case job, ok := <-w.jobQueue:
				if !ok { // Channel closed
					log.Printf("Worker %d: Job queue closed, stopping.", w.id)
					return
				}
				w.performCrawl(job)
			case <-w.quit:
				log.Printf("Worker %d: Stopping", w.id)
				return
			}
		}
	}()
}

// performCrawl executes the actual crawling logic using the main Crawler's CrawlURL method.
func (w *Worker) performCrawl(job core.CrawlJob) {
	log.Printf("Worker %d: Crawling %s", w.id, job.URL)
	result := w.crawler.CrawlURL(job)
	w.resultChan <- result
}

// Stop signals the worker to stop processing jobs.
func (w *Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}
