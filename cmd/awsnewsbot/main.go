// Package main is the entrypoint for the AWS news bot Lambda.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/thulasirajkomminar/aws-news-bot/internal/app/awsnews"
	"github.com/thulasirajkomminar/aws-news-bot/internal/config"
	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/dynamodb"
)

// Version is set at build time via -ldflags "-X main.Version=...".
//
//nolint:gochecknoglobals // populated by linker via -X ldflag, must be a var
var Version string

func main() {
	configureLogging()
	log.Info().Str("version", Version).Msg("starting AWS news bot")

	ctx := context.Background()

	awscfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("error loading AWS config")
	}

	cfg, err := config.New(ctx, &awscfg)
	if err != nil {
		log.Fatal().Err(err).Msg("error loading config")
	}

	store := dynamodb.NewStore(&awscfg, cfg.DynamoDB.TableName)
	service := awsnews.NewService(cfg, store)

	lambda.Start(func(ctx context.Context) error {
		attachRequestID(ctx)

		err := service.ProcessFeeds(ctx)
		if err == nil {
			return nil
		}

		return fmt.Errorf("processing feeds: %w", err)
	})
}

func configureLogging() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	level := strings.ToLower(os.Getenv("LOG_LEVEL"))
	if level == "" {
		return
	}

	parsed, err := zerolog.ParseLevel(level)
	if err != nil {
		log.Warn().Err(err).Str("LOG_LEVEL", level).Msg("invalid log level, keeping default")

		return
	}

	zerolog.SetGlobalLevel(parsed)
}

func attachRequestID(ctx context.Context) {
	lc, ok := lambdacontext.FromContext(ctx)
	if !ok {
		return
	}

	log.Logger = log.With().Str("awsRequestID", lc.AwsRequestID).Logger()
}
