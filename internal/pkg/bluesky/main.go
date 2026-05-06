// Package bluesky publishes AWS news items to a Bluesky account.
package bluesky

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

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
	bodySeparator        = "\n\n"
	maxPostLength        = 300
	truncationSuffix     = "..."
	numBodySeparators    = 2
	rkeyHashBytes        = 8
)

// Client is an authenticated Bluesky XRPC client.
type Client struct {
	client *xrpc.Client
}

// NewClient authenticates against the Bluesky PDS and returns a Client.
func NewClient(ctx context.Context, identifier, password string) (*Client, error) {
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

// Post creates a Bluesky post for item under handle's repo, using key as a
// stable identifier so re-attempts for the same logical item are detected
// server-side as duplicates and reported as success (preventing duplicate
// posts when DynamoDB persistence fails after a successful post).
func (b *Client) Post(ctx context.Context, handle, key string, item *rss.NewsItem) error {
	tags := generateTags(item.Categories)
	postText, facets := buildPost(item, tags)
	rkey := rkeyFor(key)

	post := &bsky.FeedPost{
		LexiconTypeID: collectionName,
		Text:          postText,
		CreatedAt:     time.Now().Format(time.RFC3339Nano),
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
		Rkey:       &rkey,
		Record:     &lexutil.LexiconTypeDecoder{Val: post},
	})
	if err != nil {
		if isDuplicateRecordErr(err) {
			return nil
		}

		return fmt.Errorf("creating Bluesky record: %w", err)
	}

	return nil
}

// rkeyFor derives a Bluesky-safe record key from an arbitrary stable
// identifier (Rkeys must match a strict charset; truncated SHA-256 hex
// satisfies it and gives ~64 bits of collision resistance).
func rkeyFor(key string) string {
	sum := sha256.Sum256([]byte(key))

	return hex.EncodeToString(sum[:rkeyHashBytes])
}

func isDuplicateRecordErr(err error) bool {
	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "recordalreadyexists") ||
		strings.Contains(msg, "duplicate")
}

// buildPost emits text and facets together so facet offsets point at the
// trailing tag block, not at any hashtag-shaped substring inside the body.
// Length is counted in runes (a closer approximation of Bluesky's grapheme
// limit than byte length).
func buildPost(item *rss.NewsItem, tags []string) (string, []*bsky.RichtextFacet) {
	tagsJoined := strings.Join(tags, tagSeparator)

	tagsRunes := utf8.RuneCountInString(tagsJoined)
	separatorRunes := utf8.RuneCountInString(bodySeparator) * numBodySeparators
	titleBudget := maxPostLength - tagsRunes - separatorRunes

	title := truncateRunes(item.Title, titleBudget, truncationSuffix)
	titleRunes := utf8.RuneCountInString(title)

	descriptionBudget := titleBudget - titleRunes
	description := truncateRunes(item.Description, descriptionBudget, truncationSuffix)

	var sb strings.Builder

	sb.WriteString(title)
	sb.WriteString(bodySeparator)
	sb.WriteString(description)
	sb.WriteString(bodySeparator)

	tagBlockStart := sb.Len()
	sb.WriteString(tagsJoined)

	return sb.String(), tagFacets(tags, tagBlockStart)
}

func tagFacets(tags []string, tagBlockStart int) []*bsky.RichtextFacet {
	facets := make([]*bsky.RichtextFacet, 0, len(tags))
	cursor := tagBlockStart

	for _, tag := range tags {
		end := cursor + len(tag)
		facets = append(facets, &bsky.RichtextFacet{
			Index: &bsky.RichtextFacet_ByteSlice{
				ByteStart: int64(cursor),
				ByteEnd:   int64(end),
			},
			Features: []*bsky.RichtextFacet_Features_Elem{
				{
					RichtextFacet_Tag: &bsky.RichtextFacet_Tag{
						LexiconTypeID: richtextFacetTagType,
						Tag:           strings.TrimPrefix(tag, "#"),
					},
				},
			},
		})

		cursor = end + len(tagSeparator)
	}

	return facets
}

func truncateRunes(s string, budget int, suffix string) string {
	if budget <= 0 {
		return ""
	}

	if utf8.RuneCountInString(s) <= budget {
		return s
	}

	suffixRunes := utf8.RuneCountInString(suffix)
	if budget <= suffixRunes {
		return string([]rune(suffix)[:budget])
	}

	keep := budget - suffixRunes

	return string([]rune(s)[:keep]) + suffix
}

func generateTags(categories []string) []string {
	tags := []string{awsRootTag}
	seen := map[string]struct{}{awsRootTag: {}}

	for _, category := range categories {
		for subCategory := range strings.SplitSeq(category, ",") {
			subCategory = strings.TrimSpace(subCategory)

			tag, ok := categoryToTag(subCategory)
			if !ok {
				continue
			}

			if _, dup := seen[tag]; dup {
				continue
			}

			seen[tag] = struct{}{}

			tags = append(tags, tag)
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
