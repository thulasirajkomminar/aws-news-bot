// Package awsnews orchestrates fetching AWS RSS feeds and posting to Bluesky.
package awsnews

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/bluesky"
	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/rss"
)

const (
	blueskyPostInterval = time.Second
	// maxPostsPerRun caps work per Lambda invocation so a large backlog
	// (e.g. on first deploy) cannot exceed the function's timeout. Remaining
	// items are picked up on the next scheduled run.
	maxPostsPerRun = 50
)

func (s *Service) processRSSFeed(ctx context.Context, bsky *bluesky.Client, feedURL, suffix string) error {
	rssFeed := rss.NewParser()

	newsItems, err := rssFeed.ScrapeFeed(ctx, feedURL)
	if err != nil {
		log.Error().Err(err).Str("url", feedURL).Msg("error parsing RSS feed")

		return fmt.Errorf("scraping feed %s: %w", feedURL, err)
	}

	keys := make([]string, len(newsItems))
	for i := range newsItems {
		keys[i] = newsItems[i].GUID + suffix
	}

	published, err := s.db.IsPublishedBatch(ctx, keys)
	if err != nil {
		return fmt.Errorf("checking publish status: %w", err)
	}

	rateLimiter := time.NewTicker(blueskyPostInterval)
	defer rateLimiter.Stop()

	return s.postAll(ctx, bsky, newsItems, suffix, published, rateLimiter)
}

func (s *Service) postAll(
	ctx context.Context,
	bsky *bluesky.Client,
	items []rss.NewsItem,
	suffix string,
	published map[string]bool,
	rateLimiter *time.Ticker,
) error {
	posted := 0

	for i := range items {
		item := &items[i]
		if published[item.GUID+suffix] {
			continue
		}

		if posted >= maxPostsPerRun {
			log.Info().
				Int("posted", posted).
				Int("scanned", i+1).
				Int("total", len(items)).
				Msg("reached max posts per run, deferring rest")

			break
		}

		err := waitForTick(ctx, rateLimiter, posted)
		if err != nil {
			return err
		}

		s.postToBluesky(ctx, bsky, item, suffix)

		posted++
	}

	return nil
}

func waitForTick(ctx context.Context, rateLimiter *time.Ticker, n int) error {
	if n == 0 {
		return nil
	}

	select {
	case <-rateLimiter.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	}
}
