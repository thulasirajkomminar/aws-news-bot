package bluesky

import (
	"context"
	"strings"
	"time"

	"github.com/komminarlabs/aws-news/internal/rss"
	"github.com/reiver/go-atproto/com/atproto/repo"
	"github.com/reiver/go-atproto/com/atproto/server"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	collectionName = "app.bsky.feed.post"
	maxPostLength  = 300
)

type Bluesky interface {
	Post(ctx context.Context, handle string, item rss.NewsItem) error
}

type blueskyImpl struct {
	bearerToken string
}

func NewBluesky(identifier, password string) (Bluesky, error) {
	var dst server.CreateSessionResponse

	err := server.CreateSession(&dst, identifier, password)
	if err != nil {
		return nil, err
	}

	return &blueskyImpl{
		bearerToken: dst.AccessJWT,
	}, nil
}

func (b *blueskyImpl) Post(ctx context.Context, handle string, item rss.NewsItem) error {
	var dst repo.CreateRecordResponse
	when := time.Now().Format(time.RFC3339)
	postText := constructPostText(item)
	tags := generateTags(item.Categories)
	facets := constructFacets(postText, tags)

	post := map[string]any{
		"$type":     collectionName,
		"text":      postText,
		"createdAt": when,
		"facets":    facets,
		"embed": map[string]any{
			"$type": "app.bsky.embed.external",
			"external": map[string]any{
				"uri":         item.Link,
				"title":       item.Title,
				"description": item.Description,
			},
		},
	}

	err := repo.CreateRecord(&dst, b.bearerToken, handle, collectionName, post)
	if err != nil {
		return err
	}
	return nil
}

func constructPostText(item rss.NewsItem) string {
	postDescription := "\n\n"
	tags := generateTags(item.Categories)
	remainingLength := maxPostLength - (len(item.Title+"\n") + len(strings.Join(tags, " ")))

	if remainingLength > 0 && len(item.Description) > remainingLength {
		postDescription += item.Description[:remainingLength-8] + "..."
	} else if remainingLength > 0 {
		postDescription += item.Description
	}
	return item.Title + postDescription + "\n\n" + strings.Join(tags, " ")
}

func generateTags(categories []string) []string {
	tags := []string{"#AWS"}

	for _, category := range categories {
		// Split the category string by commas
		subCategories := strings.Split(category, ",")
		for _, subCategory := range subCategories {
			subCategory = strings.TrimSpace(subCategory) // Trim any leading/trailing whitespace
			if strings.HasPrefix(subCategory, "general:products/") {
				tag := strings.ReplaceAll(subCategory, "general:products/", "")
				tag = strings.ReplaceAll(tag, "-", " ")
				tag = toCamelCase(tag)
				tags = append(tags, "#"+tag)
			} else if !strings.Contains(subCategory, ":") && !strings.Contains(subCategory, "/") {
				tag := strings.ReplaceAll(subCategory, "-", " ")
				tag = toCamelCase(tag)
				tags = append(tags, "#"+tag)
			}
		}
	}
	return tags
}

func toCamelCase(str string) string {
	words := strings.Fields(str)
	caser := cases.Title(language.English)
	for i := range words {
		words[i] = caser.String(words[i])
	}
	return strings.Join(words, "")
}

func constructFacets(postText string, tags []string) []map[string]any {
	var facets []map[string]any
	for _, tag := range tags {
		start := strings.Index(postText, tag)
		if start != -1 {
			end := start + len(tag)
			facet := map[string]any{
				"index": map[string]int{
					"byteStart": start,
					"byteEnd":   end,
				},
				"features": []map[string]any{
					{
						"$type": "app.bsky.richtext.facet#tag",
						"tag":   tag[1:], // Remove the '#' from the tag
					},
				},
			}
			facets = append(facets, facet)
		}
	}
	return facets
}
