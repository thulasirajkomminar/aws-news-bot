// Package bluesky publishes AWS news items to a Bluesky account.
package bluesky

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/rss"
)

const (
	defaultPDSHost       = "https://bsky.social"
	collectionName       = "app.bsky.feed.post"
	embedExternalType    = "app.bsky.embed.external"
	richtextFacetTagType = "app.bsky.richtext.facet#tag"
	productsPrefix       = "general:products/"
	awsRootTag           = "#AWS"
	tagSeparator         = " "
	maxPostLength        = 300
	truncationSuffix     = "..."
	truncationOverhead   = 8
)

// Bluesky is the contract for posting news items to Bluesky.
type Bluesky interface {
	Post(ctx context.Context, handle string, item *rss.NewsItem) error
}

// Client is an authenticated Bluesky XRPC client.
type Client struct {
	client *xrpc.Client
}

// NewBluesky authenticates against the Bluesky PDS and returns a Client.
func NewBluesky(ctx context.Context, identifier, password string) (*Client, error) {
	client := &xrpc.Client{Host: defaultPDSHost}

	out, err := atproto.ServerCreateSession(ctx, client, &atproto.ServerCreateSession_Input{
		Identifier: identifier,
		Password:   password,
	})
	if err != nil {
		return nil, fmt.Errorf("creating Bluesky session: %w", err)
	}

	client.Auth = &xrpc.AuthInfo{
		AccessJwt:  out.AccessJwt,
		RefreshJwt: out.RefreshJwt,
		Did:        out.Did,
		Handle:     out.Handle,
	}

	return &Client{client: client}, nil
}

// Post creates a Bluesky post for the given news item under handle's repo.
func (b *Client) Post(ctx context.Context, handle string, item *rss.NewsItem) error {
	postText := constructPostText(item)
	tags := generateTags(item.Categories)
	facets := constructFacets(postText, tags)

	post := &bsky.FeedPost{
		LexiconTypeID: collectionName,
		Text:          postText,
		CreatedAt:     time.Now().Format(time.RFC3339),
		Facets:        facets,
		Embed: &bsky.FeedPost_Embed{
			EmbedExternal: &bsky.EmbedExternal{
				LexiconTypeID: embedExternalType,
				External: &bsky.EmbedExternal_External{
					Uri:         item.Link,
					Title:       item.Title,
					Description: item.Description,
				},
			},
		},
	}

	_, err := atproto.RepoCreateRecord(ctx, b.client, &atproto.RepoCreateRecord_Input{
		Collection: collectionName,
		Repo:       handle,
		Record:     &lexutil.LexiconTypeDecoder{Val: post},
	})
	if err != nil {
		return fmt.Errorf("creating Bluesky record: %w", err)
	}

	return nil
}

func constructPostText(item *rss.NewsItem) string {
	postDescription := "\n\n"
	tags := generateTags(item.Categories)
	remainingLength := maxPostLength - (len(item.Title+"\n") + len(strings.Join(tags, tagSeparator)))

	switch {
	case remainingLength > 0 && len(item.Description) > remainingLength:
		postDescription += item.Description[:remainingLength-truncationOverhead] + truncationSuffix
	case remainingLength > 0:
		postDescription += item.Description
	default:
	}

	return item.Title + postDescription + "\n\n" + strings.Join(tags, tagSeparator)
}

func generateTags(categories []string) []string {
	tags := []string{awsRootTag}

	for _, category := range categories {
		for subCategory := range strings.SplitSeq(category, ",") {
			subCategory = strings.TrimSpace(subCategory)
			if tag, ok := categoryToTag(subCategory); ok {
				tags = append(tags, tag)
			}
		}
	}

	return tags
}

func categoryToTag(subCategory string) (string, bool) {
	switch {
	case strings.HasPrefix(subCategory, productsPrefix):
		raw := strings.ReplaceAll(subCategory, productsPrefix, "")

		return "#" + toCamelCase(strings.ReplaceAll(raw, "-", tagSeparator)), true
	case !strings.Contains(subCategory, ":") && !strings.Contains(subCategory, "/"):
		return "#" + toCamelCase(strings.ReplaceAll(subCategory, "-", tagSeparator)), true
	default:
		return "", false
	}
}

func toCamelCase(str string) string {
	words := strings.Fields(str)

	caser := cases.Title(language.English)
	for i := range words {
		words[i] = caser.String(words[i])
	}

	return strings.Join(words, "")
}

func constructFacets(postText string, tags []string) []*bsky.RichtextFacet {
	var facets []*bsky.RichtextFacet

	for _, tag := range tags {
		start := strings.Index(postText, tag)
		if start == -1 {
			continue
		}

		end := start + len(tag)
		facets = append(facets, &bsky.RichtextFacet{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: int64(start),
				ByteEnd:   int64(end),
			},
			Features: []*bsky.RichtextFacet_Features_Elem{
				{
					RichtextFacet_Tag: &bsky.RichtextFacet_Tag{
						LexiconTypeID: richtextFacetTagType,
						Tag:           tag[1:],
					},
				},
			},
		})
	}

	return facets
}
