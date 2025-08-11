package crawler

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Extractor is responsible for extracting and cleaning content from HTML.
type Extractor struct{}

// NewExtractor creates a new Extractor.
func NewExtractor() *Extractor {
	return &Extractor{}
}

// ExtractContent extracts title, description, body text, and links from HTML using goquery.
func (e *Extractor) ExtractContent(htmlContent []byte) (title, description, bodyText string, links []string, contentHash string, err error) {
	reader := bytes.NewReader(htmlContent)
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return "", "", "", nil, "", fmt.Errorf("failed to parse HTML with goquery: %w", err)
	}

	title = e.extractTitle(doc)
	description = e.extractMetaDescription(doc)
	bodyText = e.extractBodyText(doc)
	links = e.extractLinks(doc)
	contentHash = e.generateContentHash(htmlContent)

	return title, description, bodyText, links, contentHash, nil
}

func (e *Extractor) extractTitle(doc *goquery.Document) string {
	return strings.TrimSpace(doc.Find("title").First().Text())
}

func (e *Extractor) extractMetaDescription(doc *goquery.Document) string {
	content, _ := doc.Find("meta[name=description]").First().Attr("content")
	return strings.TrimSpace(content)
}

func (e *Extractor) extractBodyText(doc *goquery.Document) string {
	// Remove script and style tags to avoid extracting their content
	doc.Find("script, style").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	// Try to find the main content in semantic tags, fall back to body
	var text string
	mainContent := doc.Find("main")
	if mainContent.Length() > 0 {
		text = mainContent.Text()
	} else {
		text = doc.Find("body").Text()
	}

	// Normalize whitespace
	return strings.Join(strings.Fields(text), " ")
}

func (e *Extractor) extractLinks(doc *goquery.Document) []string {
	var links []string
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			links = append(links, strings.TrimSpace(href))
		}
	})
	return links
}

func (e *Extractor) generateContentHash(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf("%x", hash)
}