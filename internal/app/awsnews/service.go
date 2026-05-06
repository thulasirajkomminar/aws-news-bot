package awsnews

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/thulasirajkomminar/aws-news-bot/internal/config"
	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/bluesky"
	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/dynamodb"
)

// Service represents the AWS News service.
type Service struct {
	cfg *config.Config
	db  *dynamodb.Store
}

// NewService creates a new AWS News service instance.
func NewService(cfg *config.Config, db *dynamodb.Store) *Service {
	return &Service{cfg: cfg, db: db}
}

// ProcessFeeds re-authenticates Bluesky on every invocation (warm Lambda
// containers can outlive the ~2h session JWT) and runs each feed
// independently, joining any errors so one feed's outage doesn't silence
// the other.
func (s *Service) ProcessFeeds(ctx context.Context) error {
	bsky, err := bluesky.NewClient(ctx, s.cfg.Bluesky.Handle, s.cfg.Bluesky.Password)
	if err != nil {
		log.Error().Err(err).Msg("error creating Bluesky client")

		return fmt.Errorf("creating Bluesky client: %w", err)
	}

	feeds := []struct {
		url    string
		suffix string
		label  string
	}{
		{s.cfg.WhatsNewRSSFeed.URL, "-whatsnew", "whatsnew"},
		{s.cfg.NewsBlogRSSFeed.URL, "-newsblog", "newsblog"},
	}

	var errs []error

	for _, feed := range feeds {
		err := s.processRSSFeed(ctx, bsky, feed.url, feed.suffix)
		if err != nil {
			log.Error().Err(err).Str("feed", feed.label).Msg("error processing feed")

			errs = append(errs, fmt.Errorf("%s: %w", feed.label, err))
		}
	}

	return errors.Join(errs...)
}
