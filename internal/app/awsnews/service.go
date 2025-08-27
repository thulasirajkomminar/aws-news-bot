package awsnews

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/thulasirajkomminar/aws-news-bot/internal/config"
	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/bluesky"
	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/dynamodb"
)

// Service represents the AWS News service
type Service struct {
	cfg  *config.Config
	db   dynamodb.DynamoDB
	bsky bluesky.Bluesky
}

// NewService creates a new AWS News service instance
func NewService(ctx context.Context, cfg *config.Config) (*Service, error) {
	db, err := dynamodb.NewDynamoDB(ctx, cfg.DynamoDB.TableName)
	if err != nil {
		log.Error().Err(err).Msg("error creating DynamoDB client")
		return nil, err
	}

	bsky, err := bluesky.NewBluesky(cfg.Bluesky.Handle, cfg.Bluesky.Password)
	if err != nil {
		log.Error().Err(err).Msg("error creating Bluesky client")
		return nil, err
	}

	return &Service{
		cfg:  cfg,
		db:   db,
		bsky: bsky,
	}, nil
}

// ProcessFeeds processes all RSS feeds and posts items to Bluesky
func (s *Service) ProcessFeeds(ctx context.Context) error {
	// Add reasonable timeout for RSS feed scraping
	scrapeCtx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()

	err := s.processRSSFeed(scrapeCtx, s.cfg.WhatsNewRSSFeed.URL, "-whatsnew")
	if err != nil {
		log.Error().Err(err).Msg("error scraping AWS what's new feed")
		return err
	}

	err = s.processRSSFeed(scrapeCtx, s.cfg.NewsBlogRSSFeed.URL, "-newsblog")
	if err != nil {
		log.Error().Err(err).Msg("error scraping AWS news blog feed")
		return err
	}

	return nil
}
