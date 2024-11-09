package rss

import (
	"context"

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
	Description string
	GUID        string
	Link        string
	Title       string
}

func (f *feedImpl) ScrapeAWSNews(ctx context.Context, url string) ([]NewsItem, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil, err
	}

	var newsItems []NewsItem
	for _, item := range feed.Items {
		newsItems = append(newsItems, NewsItem{
			Description: html2text.HTML2Text(item.Description),
			GUID:        item.GUID,
			Link:        item.Link,
			Title:       item.Title,
		})
	}
	return newsItems, nil
}
