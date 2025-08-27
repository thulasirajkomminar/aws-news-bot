package main

import (
	"context"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rs/zerolog/log"
	"github.com/thulasirajkomminar/aws-news-bot/internal/app/awsnews"
	"github.com/thulasirajkomminar/aws-news-bot/internal/config"
)

var service *awsnews.Service

func Handler(ctx context.Context) error {
	return service.ProcessFeeds(ctx)
}

func main() {
	ctx := context.Background()

	cfg, err := config.New(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("error loading config")
	}

	service, err = awsnews.NewService(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("error creating AWS News service")
	}

	lambda.Start(Handler)
}
