// Package dynamodb provides DynamoDB-backed storage for publish status.
package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const ttlDuration = 30 * 24 * time.Hour

// PublishStatus is a record stored in DynamoDB to track Bluesky posts.
type PublishStatus struct {
	ExpiresAt int64 `dynamodbav:"ExpiresAt"`
	GUID      string
	Published string
	Title     string
}

// DynamoDB is the contract for the publish-status store.
type DynamoDB interface {
	StorePublishStatus(ctx context.Context, status PublishStatus) error
	IsPublished(ctx context.Context, guid string) (bool, error)
}

// Store is a DynamoDB-backed implementation of DynamoDB.
type Store struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoDB constructs a Store backed by the default AWS configuration.
func NewDynamoDB(ctx context.Context, tableName string) (*Store, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading default AWS config: %w", err)
	}

	return &Store{
		client:    dynamodb.NewFromConfig(cfg),
		tableName: tableName,
	}, nil
}

// StorePublishStatus writes a PublishStatus record with a TTL.
func (db *Store) StorePublishStatus(ctx context.Context, status PublishStatus) error {
	status.ExpiresAt = time.Now().Add(ttlDuration).Unix()

	av, err := attributevalue.MarshalMap(status)
	if err != nil {
		return fmt.Errorf("marshalling publish status: %w", err)
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(db.tableName),
	}

	_, err = db.client.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("putting item to DynamoDB: %w", err)
	}

	return nil
}

// IsPublished reports whether a record with the given GUID exists.
func (db *Store) IsPublished(ctx context.Context, guid string) (bool, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(db.tableName),
		Key: map[string]types.AttributeValue{
			"GUID": &types.AttributeValueMemberS{
				Value: guid,
			},
		},
	}

	result, err := db.client.GetItem(ctx, input)
	if err != nil {
		return false, fmt.Errorf("getting item from DynamoDB: %w", err)
	}

	return result.Item != nil, nil
}
