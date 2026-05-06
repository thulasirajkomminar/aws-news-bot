// Package awsnews orchestrates fetching AWS RSS feeds and posting to Bluesky.
package awsnews

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/rss"
)

const blueskyPostInterval = time.Second

// processRSSFeed processes an RSS feed and posts items to Bluesky.
func (s *Service) processRSSFeed(ctx context.Context, feedURL string, suffix string) error {
	rssFeed := rss.NewFeed()

	newsItems, err := rssFeed.ScrapeFeed(ctx, feedURL)
	if err != nil {
		log.Error().Err(err).Msg("error parsing RSS feed")

		return fmt.Errorf("scraping feed %s: %w", feedURL, err)
	}

	rateLimiter := time.NewTicker(blueskyPostInterval)
	defer rateLimiter.Stop()

	for _, item := range newsItems {
		err := s.processItem(ctx, &item, suffix, rateLimiter)
		if err != nil {
			return err
		}
	}

	return nil
}

// processItem publishes a single RSS item if it has not been posted yet,
// returning an error only when the context is cancelled.
func (s *Service) processItem(ctx context.Context, item *rss.NewsItem, suffix string, rateLimiter *time.Ticker) error {
	isPublished, err := s.db.IsPublished(ctx, item.GUID+suffix)
	if err != nil {
		log.Warn().Err(err).Msg("error checking publish status")

		return nil
	}

	if isPublished {
		return nil
	}

	select {
	case <-rateLimiter.C:
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	}

	_ = s.postToBluesky(ctx, item, suffix)

	return nil
}
