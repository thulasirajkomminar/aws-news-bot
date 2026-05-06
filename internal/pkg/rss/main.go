// Package rss parses AWS RSS feeds into NewsItem records.
package rss

import (
	"context"
	"errors"
	"fmt"

	"github.com/k3a/html2text"
	"github.com/mmcdole/gofeed"
)

// ErrNoNewsItems is returned when a feed contains no usable items.
var ErrNoNewsItems = errors.New("no valid news items found in feed")

// Parser scrapes RSS feeds using gofeed.
type Parser struct{}

// NewParser returns a new Parser.
func NewParser() *Parser {
	return &Parser{}
}

// NewsItem describes a single AWS news item.
type NewsItem struct {
	Categories  []string
	Description string
	GUID        string
	Link        string
	Published   string
	Title       string
}

// ScrapeFeed downloads the RSS feed at url and returns its valid items.
func (f *Parser) ScrapeFeed(ctx context.Context, url string) ([]NewsItem, error) {
	fp := gofeed.NewParser()

	feed, err := fp.ParseURLWithContext(url, ctx)
	if err != nil {
		return nil, fmt.Errorf("parsing RSS feed: %w", err)
	}

	newsItems := extractNewsItems(feed.Items)
	if len(newsItems) == 0 {
		return nil, ErrNoNewsItems
	}

	return newsItems, nil
}

func extractNewsItems(items []*gofeed.Item) []NewsItem {
	newsItems := make([]NewsItem, 0, len(items))

	for _, item := range items {
		if item.Title == "" || item.Link == "" {
			continue
		}

		newsItems = append(newsItems, NewsItem{
			Categories:  item.Categories,
			Description: html2text.HTML2Text(item.Description),
			GUID:        item.GUID,
			Link:        item.Link,
			Published:   publishedString(item),
			Title:       item.Title,
		})
	}

	return newsItems
}

// publishedString falls back to the raw <pubDate> text when gofeed cannot
// parse it (PublishedParsed is nil for unrecognised date formats).
func publishedString(item *gofeed.Item) string {
	if item.PublishedParsed != nil {
		return item.PublishedParsed.String()
	}

	return item.Published
}
