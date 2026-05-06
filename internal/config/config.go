// Package config loads runtime configuration from env vars and SSM.
package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
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
// Bluesky password from SSM Parameter Store.
func New(ctx context.Context) (*Config, error) {
	cfg := Config{}
	opts := env.Options{RequiredIfNoDef: true}

	err := env.ParseWithOptions(&cfg, opts)
	if err != nil {
		return nil, fmt.Errorf("parsing environment variables: %w", err)
	}

	awscfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading default AWS config: %w", err)
	}

	if cfg.Bluesky.PasswordPath == "" {
		return nil, ErrMissingPasswordPath
	}

	ssmClient := ssm.NewFromConfig(awscfg)
	decryption := true

	blueskyPasswordParam, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &cfg.Bluesky.PasswordPath,
		WithDecryption: &decryption,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching Bluesky password from SSM: %w", err)
	}

	cfg.Bluesky.Password = *blueskyPasswordParam.Parameter.Value

	return &cfg, nil
}
