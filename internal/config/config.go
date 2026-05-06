// Package config loads runtime configuration from env vars and SSM.
package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/caarlos0/env/v11"
)

// ErrMissingPasswordPath is returned when BLUESKY_PASSWORD_PATH is unset.
var ErrMissingPasswordPath = errors.New("BLUESKY_PASSWORD_PATH environment variable not set")

// Bluesky holds Bluesky credentials and SSM lookup metadata.
type Bluesky struct {
	Handle       string `env:"BLUESKY_HANDLE"`
	Password     string
	PasswordPath string `env:"BLUESKY_PASSWORD_PATH"`
}

// DynamoDB holds DynamoDB connection settings.
type DynamoDB struct {
	TableName string `env:"DYNAMODB_TABLE_NAME"`
}

// NewsBlogRSSFeed holds the AWS News Blog feed URL.
type NewsBlogRSSFeed struct {
	URL string `env:"NEWSBLOG_RSSFEED_URL"`
}

// WhatsNewRSSFeed holds the AWS What's New feed URL.
type WhatsNewRSSFeed struct {
	URL string `env:"WHATSNEW_RSSFEED_URL"`
}

// Config aggregates all runtime configuration for the bot.
type Config struct {
	Bluesky         Bluesky
	DynamoDB        DynamoDB
	NewsBlogRSSFeed NewsBlogRSSFeed
	WhatsNewRSSFeed WhatsNewRSSFeed
}

// New parses configuration from environment variables and resolves the
// Bluesky password from SSM Parameter Store using the supplied AWS config.
func New(ctx context.Context, awscfg *aws.Config) (*Config, error) {
	cfg := Config{}

	err := env.ParseWithOptions(&cfg, env.Options{RequiredIfNoDef: true})
	if err != nil {
		return nil, fmt.Errorf("parsing environment variables: %w", err)
	}

	if cfg.Bluesky.PasswordPath == "" {
		return nil, ErrMissingPasswordPath
	}

	ssmClient := ssm.NewFromConfig(*awscfg)

	param, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &cfg.Bluesky.PasswordPath,
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("fetching Bluesky password from SSM: %w", err)
	}

	cfg.Bluesky.Password = *param.Parameter.Value

	return &cfg, nil
}
