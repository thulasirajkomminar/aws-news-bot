package config

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/caarlos0/env/v11"
)

type Bluesky struct {
	Handle       string `env:"BLUESKY_HANDLE"`
	Password     string
	PasswordPath string `env:"BLUESKY_PASSWORD_PATH"`
}

type DynamoDB struct {
	TableName string `env:"DYNAMODB_TABLE_NAME"`
}

type NewsBlogRSSFeed struct {
	URL string `env:"NEWSBLOG_RSSFEED_URL"`
}

type WhatsNewRSSFeed struct {
	URL string `env:"WHATSNEW_RSSFEED_URL"`
}

type Config struct {
	Bluesky         Bluesky
	DynamoDB        DynamoDB
	NewsBlogRSSFeed NewsBlogRSSFeed
	WhatsNewRSSFeed WhatsNewRSSFeed
}

func New(ctx context.Context) (*Config, error) {
	cfg := Config{}
	opts := env.Options{RequiredIfNoDef: true}

	err := env.ParseWithOptions(&cfg, opts)
	if err != nil {
		return nil, err
	}

	awscfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	ssmClient := ssm.NewFromConfig(awscfg)
	decryption := true

	if cfg.Bluesky.PasswordPath == "" {
		return nil, fmt.Errorf("BLUESKY_PASSWORD_PATH environment variable not set")
	}

	blueskyPasswordParam, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           &cfg.Bluesky.PasswordPath,
		WithDecryption: &decryption,
	})
	if err != nil {
		return nil, err
	}
	cfg.Bluesky.Password = *blueskyPasswordParam.Parameter.Value

	return &cfg, nil
}
