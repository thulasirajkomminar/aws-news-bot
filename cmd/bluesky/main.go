package main

import (
	"context"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/komminarlabs/aws-news/internal/bluesky"
	"github.com/komminarlabs/aws-news/internal/config"
	"github.com/komminarlabs/aws-news/internal/dynamodb"
	"github.com/komminarlabs/aws-news/internal/rss"
	"github.com/rs/zerolog/log"
)

func Handler(ctx context.Context, event events.KinesisEvent) error {
	cfg, err := config.New(ctx)
	if err != nil {
		log.Logger.Error().Err(err).Msg("error loading config")
		return err
	}

	// Add reasonable timeout for RSS feed scraping
	scrapeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rssFeed := rss.NewFeed()
	newsItems, err := rssFeed.ScrapeAWSNews(scrapeCtx, cfg.RSSFeed.URL)
	if err != nil {
		log.Logger.Error().Err(err).Msg("error parsing RSS feed")
		return err
	}

	db, err := dynamodb.NewDynamoDB(ctx, cfg.DynamoDB.TableName)
	if err != nil {
		log.Logger.Error().Err(err).Msg("error creating DynamoDB client")
		return err
	}

	bsky, err := bluesky.NewBluesky(cfg.Bluesky.Handle, cfg.Bluesky.Password)
	if err != nil {
		log.Logger.Error().Err(err).Msg("error creating Bluesky client")
		return err
	}

	for _, item := range newsItems {
		isPublished, err := db.IsPublished(ctx, item.GUID)
		if err != nil {
			log.Logger.Warn().Err(err).Msg("error checking publish status")
			continue
		}

		if !isPublished {
			err := bsky.Post(ctx, cfg.Bluesky.Handle, item)
			if err != nil {
				log.Logger.Warn().Err(err).Msg("error posting to Bluesky")
				continue
			}

			status := dynamodb.PublishStatus{
				GUID:  item.GUID,
				Title: item.Title,
			}
			err = db.StorePublishStatus(ctx, status)
			if err != nil {
				log.Logger.Warn().Err(err).Msg("error storing publish status")
			} else {
				log.Logger.Debug().Msg("stored publish status for: " + item.Title)
			}
		}
	}
	return nil
}

func main() {
	lambda.Start(Handler)
}
