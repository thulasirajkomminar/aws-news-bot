package awsnews

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/rss"
)

// processRSSFeed processes an RSS feed and posts items to Bluesky.
func (s *Service) processRSSFeed(ctx context.Context, feedURL string, suffix string) error {
	rssFeed := rss.NewFeed()
	newsItems, err := rssFeed.ScrapeFeed(ctx, feedURL)
	if err != nil {
		log.Error().Err(err).Msg("error parsing RSS feed")
		return err
	}

	// Create rate limiter for Bluesky API calls
	rateLimiter := time.NewTicker(time.Second) // 1 post per second
	defer rateLimiter.Stop()

	for _, item := range newsItems {
		isPublished, err := s.db.IsPublished(ctx, item.GUID+suffix)
		if err != nil {
			log.Warn().Err(err).Msg("error checking publish status")
			continue
		}

		if !isPublished {
			select {
			case <-rateLimiter.C:
				// Continue with the post
			case <-ctx.Done():
				return ctx.Err()
			}

			err := s.postToBluesky(ctx, item, suffix)
			if err != nil {
				continue
			}
		}
	}
	return nil
}
