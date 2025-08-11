package crawler

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"breeze-search/core"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// Worker is a struct that represents a crawler worker.
type Worker struct {
	id             int
	config         *CrawlerConfig
	jobQueue       <-chan core.CrawlJob
	resultChan     chan<- core.CrawlResult
	quit           chan bool
	extractor      *Extractor
	robotsHandler  *RobotsTxtHandler
	wg             *sync.WaitGroup
	chromedpCtx    context.Context
	chromedpCancel context.CancelFunc
	allocCancel    context.CancelFunc
}

// NewWorker creates a new Worker.
func NewWorker(id int, config *CrawlerConfig, jobQueue <-chan core.CrawlJob, resultChan chan<- core.CrawlResult, extractor *Extractor, robotsHandler *RobotsTxtHandler, wg *sync.WaitGroup) *Worker {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(config.UserAgent),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, browserCancel := chromedp.NewContext(allocCtx)

	return &Worker{
		id:             id,
		config:         config,
		jobQueue:       jobQueue,
		resultChan:     resultChan,
		quit:           make(chan bool),
		extractor:      extractor,
		robotsHandler:  robotsHandler,
		wg:             wg,
		chromedpCtx:    ctx,
		chromedpCancel: browserCancel,
		allocCancel:    allocCancel,
	}
}

// Start starts the worker's job processing loop.
func (w *Worker) Start() {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for {
			select {
			case job, ok := <-w.jobQueue:
				if !ok {
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

// performCrawl executes the actual crawling logic.
func (w *Worker) performCrawl(job core.CrawlJob) {
	log.Printf("Worker %d: Crawling %s", w.id, job.URL)

	result := core.CrawlResult{
		URL:         job.URL,
		Success:     false,
		LastCrawled: time.Now(),
	}

	if !w.robotsHandler.CanFetch(job.URL) {
		result.Error = fmt.Errorf("crawling disallowed by robots.txt")
		w.resultChan <- result
		return
	}

	req, err := http.NewRequest("GET", job.URL, nil)
	if err != nil {
		result.Error = fmt.Errorf("failed to create request: %w", err)
		w.resultChan <- result
		return
	}
	req.Header.Set("User-Agent", w.config.UserAgent)

	client := &http.Client{Timeout: w.config.RequestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch initial HTML: %w", err)
		w.resultChan <- result
		return
	}
	defer resp.Body.Close()

	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("failed to read initial HTML: %w", err)
		w.resultChan <- result
		return
	}

	ctx, cancel := context.WithTimeout(w.chromedpCtx, w.config.RequestTimeout)
	defer cancel()

	var finalHTML string
	err = chromedp.Run(ctx,
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, string(html)).Do(ctx)
		}),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.OuterHTML("html", &finalHTML, chromedp.ByQuery),
	)

	if err != nil {
		result.Error = fmt.Errorf("chromedp rendering failed: %w", err)
		w.resultChan <- result
		return
	}

	title, description, bodyText, links, contentHash, err := w.extractor.ExtractContent([]byte(finalHTML))
	if err != nil {
		result.Error = fmt.Errorf("failed to extract content: %w", err)
		w.resultChan <- result
		return
	}

	result.Title = title
	result.Description = description
	result.BodyText = bodyText
	result.Links = links
	result.ContentHash = contentHash
	result.Success = true

	w.resultChan <- result
}

// Stop signals the worker to stop processing jobs.
func (w *Worker) Stop() {
	w.chromedpCancel()
	w.allocCancel()
	close(w.quit)
}