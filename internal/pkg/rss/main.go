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

// Feed is the contract for RSS feed scrapers.
type Feed interface {
	ScrapeFeed(ctx context.Context, url string) ([]NewsItem, error)
}

// Parser implements Feed using gofeed.
type Parser struct{}

// NewFeed returns a new Parser.
func NewFeed() *Parser {
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

	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parsing RSS feed: %w", err)
	}

	newsItems, err := extractNewsItems(ctx, feed.Items)
	if err != nil {
		return nil, err
	}

	if len(newsItems) == 0 {
		return nil, ErrNoNewsItems
	}

	return newsItems, nil
}

func extractNewsItems(ctx context.Context, items []*gofeed.Item) ([]NewsItem, error) {
	newsItems := make([]NewsItem, 0, len(items))

	for _, item := range items {
		if item.Title == "" || item.Link == "" {
			continue
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled while parsing feed: %w", ctx.Err())
		default:
		}

		newsItems = append(newsItems, NewsItem{
			Categories:  item.Categories,
			Description: html2text.HTML2Text(item.Description),
			GUID:        item.GUID,
			Link:        item.Link,
			Published:   item.PublishedParsed.String(),
			Title:       item.Title,
		})
	}

	return newsItems, nil
}
