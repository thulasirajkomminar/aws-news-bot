package awsnews

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/bluesky"
	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/dynamodb"
	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/rss"
)

// postToBluesky logs but does not propagate failures so one bad item does
// not abort the surrounding feed loop.
func (s *Service) postToBluesky(ctx context.Context, bsky *bluesky.Client, item *rss.NewsItem, suffix string) {
	key := item.GUID + suffix

	err := bsky.Post(ctx, s.cfg.Bluesky.Handle, key, item)
	if err != nil {
		log.Warn().
			Err(err).
			Str("title", item.Title).
			Str("link", item.Link).
			Str("suffix", suffix).
			Msg("error posting to Bluesky")

		return
	}

	status := dynamodb.PublishStatus{
		GUID:      key,
		Published: item.Published,
		Title:     item.Title,
	}

	err = s.db.StorePublishStatus(ctx, status)
	if err != nil {
		log.Warn().Err(err).Str("guid", status.GUID).Msg("error storing publish status")

		return
	}

	log.Debug().Str("title", item.Title).Msg("stored publish status")
}
