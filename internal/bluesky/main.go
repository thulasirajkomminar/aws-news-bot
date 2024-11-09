package bluesky

import (
	"context"
	"time"

	"github.com/komminarlabs/aws-news/internal/rss"
	"github.com/reiver/go-atproto/com/atproto/repo"
	"github.com/reiver/go-atproto/com/atproto/server"
)

const (
	collectionName = "app.bsky.feed.post"
	maxPostLength  = 300
)

type Bluesky interface {
	Post(ctx context.Context, handle string, item rss.NewsItem) error
}

type blueskyImpl struct {
	bearerToken *string
}

func NewBluesky(identifier, password string) (Bluesky, error) {
	var dst server.CreateSessionResponse

	err := server.CreateSession(&dst, identifier, password)
	if err != nil {
		return nil, err
	}

	return &blueskyImpl{
		bearerToken: &dst.AccessJWT,
	}, nil
}

func (b *blueskyImpl) Post(ctx context.Context, handle string, item rss.NewsItem) error {
	var dst repo.CreateRecordResponse
	when := time.Now().Format(time.RFC3339)
	postText := constructPostText(item)

	post := map[string]any{
		"$type":     collectionName,
		"text":      postText,
		"link":      "TEST, " + item.Link,
		"createdAt": when,
		"facets": []map[string]any{
			{
				"index": map[string]int{
					"byteStart": len(postText) - len(item.Link),
					"byteEnd":   len(postText),
				},
				"features": []map[string]any{
					{
						"$type": "app.bsky.richtext.facet#link",
						"uri":   item.Link,
					},
				},
			},
		},
	}

	err := repo.CreateRecord(&dst, *b.bearerToken, handle, collectionName, post)
	if err != nil {
		return err
	}
	return nil
}

func constructPostText(item rss.NewsItem) string {
	postDescription := "\n\n"
	remainingLength := maxPostLength - len(item.Title+"\n"+item.Link)

	if remainingLength > 0 && len(item.Description) > remainingLength {
		postDescription += item.Description[:remainingLength-7] + "..."
	} else if remainingLength > 0 {
		postDescription += item.Description
	}
	return item.Title + postDescription + "\n" + item.Link
}
