package rss

import (
	"context"
	"fmt"

	"github.com/k3a/html2text"
	"github.com/mmcdole/gofeed"
)

type Feed interface {
	ScrapeAWSNews(ctx context.Context, url string) ([]NewsItem, error)
}

type feedImpl struct{}

func NewFeed() Feed {
	return &feedImpl{}
}

type NewsItem struct {
	Categories  []string
	Description string
	GUID        string
	Link        string
	Title       string
}

func (f *feedImpl) ScrapeAWSNews(ctx context.Context, url string) ([]NewsItem, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed: %w", err)
	}

	// Pre-allocate slice with known capacity
	newsItems := make([]NewsItem, 0, len(feed.Items))

	for _, item := range feed.Items {
		// Skip items with missing required fields
		if item.Title == "" || item.Link == "" {
			continue
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		newsItems = append(newsItems, NewsItem{
			Categories:  item.Categories,
			Description: html2text.HTML2Text(item.Description),
			GUID:        item.GUID,
			Link:        item.Link,
			Title:       item.Title,
		})
	}

	if len(newsItems) == 0 {
		return nil, fmt.Errorf("no valid news items found in feed")
	}

	return newsItems, nil
}
