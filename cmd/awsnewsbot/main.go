// Package main is the entrypoint for the AWS news bot Lambda.
package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rs/zerolog/log"

	"github.com/thulasirajkomminar/aws-news-bot/internal/app/awsnews"
	"github.com/thulasirajkomminar/aws-news-bot/internal/config"
)

func main() {
	ctx := context.Background()

	cfg, err := config.New(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("error loading config")
	}

	service, err := awsnews.NewService(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("error creating AWS News service")
	}

	lambda.Start(func(ctx context.Context) error {
		err := service.ProcessFeeds(ctx)
		if err == nil {
			return nil
		}

		return fmt.Errorf("processing feeds: %w", err)
	})
}
