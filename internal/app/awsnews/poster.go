package awsnews

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/dynamodb"
	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/rss"
)

// postToBluesky posts an item to Bluesky and stores the publish status in DynamoDB.
func (s *Service) postToBluesky(ctx context.Context, item rss.NewsItem, suffix string) error {
	err := s.bsky.Post(ctx, s.cfg.Bluesky.Handle, item)
	if err != nil {
		log.Warn().Err(err).Msg("error posting to Bluesky: Title " + item.Title + " Link " + item.Link + " Suffix " + suffix)
		return err
	}

	status := dynamodb.PublishStatus{
		GUID:      item.GUID + suffix,
		Published: item.Published,
		Title:     item.Title,
	}
	err = s.db.StorePublishStatus(ctx, status)
	if err != nil {
		log.Warn().Err(err).Msg("error storing publish status")
	} else {
		log.Debug().Msg("stored publish status for: " + item.Title)
	}
	return nil
}
