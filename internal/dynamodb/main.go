package dynamodb

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type PublishStatus struct {
	ExpiresAt int64 `dynamodbav:"ExpiresAt"` // TTL attribute
	GUID      string
	Published string
	Title     string
}

type DynamoDB interface {
	StorePublishStatus(ctx context.Context, status PublishStatus) error
	IsPublished(ctx context.Context, guid string) (bool, error)
}

type dynamoDBImpl struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoDB(ctx context.Context, tableName string) (DynamoDB, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &dynamoDBImpl{
		client:    dynamodb.NewFromConfig(cfg),
		tableName: tableName,
	}, nil
}

func (db *dynamoDBImpl) StorePublishStatus(ctx context.Context, status PublishStatus) error {
	// Set TTL to 30 days from now
	status.ExpiresAt = time.Now().Add(30 * 24 * time.Hour).Unix()

	av, err := attributevalue.MarshalMap(status)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(db.tableName),
	}

	_, err = db.client.PutItem(ctx, input)
	return err
}

func (db *dynamoDBImpl) IsPublished(ctx context.Context, guid string) (bool, error) {
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
		return false, err
	}

	if result.Item == nil {
		return false, nil
	}
	return true, nil
}
