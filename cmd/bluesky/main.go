package main

import (
	"context"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/komminarlabs/aws-news/internal/bluesky"
	"github.com/komminarlabs/aws-news/internal/config"
	"github.com/komminarlabs/aws-news/internal/dynamodb"
	"github.com/komminarlabs/aws-news/internal/rss"
	"github.com/rs/zerolog/log"
)

// postToBluesky posts an item to Bluesky and stores the publish status in DynamoDB.
func postToBluesky(ctx context.Context, cfg *config.Config, db dynamodb.DynamoDB, bsky bluesky.Bluesky, item rss.NewsItem, suffix string) error {
	err := bsky.Post(ctx, cfg.Bluesky.Handle, item)
	if err != nil {
		log.Warn().Err(err).Msg("error posting to Bluesky")
		return err
	}

	status := dynamodb.PublishStatus{
		GUID:      item.GUID + suffix,
		Published: item.Published,
		Title:     item.Title,
	}
	err = db.StorePublishStatus(ctx, status)
	if err != nil {
		log.Warn().Err(err).Msg("error storing publish status")
	} else {
		log.Debug().Msg("stored publish status for: " + item.Title)
	}
	return nil
}

// processRSSFeed processes an RSS feed and posts items to Bluesky.
func processRSSFeed(ctx context.Context, cfg *config.Config, db dynamodb.DynamoDB, bsky bluesky.Bluesky, feedURL string, suffix string) error {
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
		isPublished, err := db.IsPublished(ctx, item.GUID+suffix)
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

			err := postToBluesky(ctx, cfg, db, bsky, item, suffix)
			if err != nil {
				continue
			}
		}
	}
	return nil
}

func Handler(ctx context.Context) error {
	cfg, err := config.New(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error loading config")
		return err
	}

	db, err := dynamodb.NewDynamoDB(ctx, cfg.DynamoDB.TableName)
	if err != nil {
		log.Error().Err(err).Msg("error creating DynamoDB client")
		return err
	}

	bsky, err := bluesky.NewBluesky(cfg.Bluesky.Handle, cfg.Bluesky.Password)
	if err != nil {
		log.Error().Err(err).Msg("error creating Bluesky client")
		return err
	}

	// Add reasonable timeout for RSS feed scraping
	scrapeCtx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()

	err = processRSSFeed(scrapeCtx, cfg, db, bsky, cfg.WhatsNewRSSFeed.URL, "-whatsnew")
	if err != nil {
		log.Error().Err(err).Msg("error scraping AWS what's new feed")
		return err
	}

	err = processRSSFeed(scrapeCtx, cfg, db, bsky, cfg.NewsBlogRSSFeed.URL, "-newsblog")
	if err != nil {
		log.Error().Err(err).Msg("error scraping AWS news blog feed")
		return err
	}
	return nil
}

func main() {
	lambda.Start(Handler)
}
