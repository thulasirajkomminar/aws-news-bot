package bluesky

import (
	"context"
	"strings"
	"time"

	atproto "github.com/bluesky-social/indigo/api/atproto"
	bsky "github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/rss"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	defaultPDSHost = "https://bsky.social"
	collectionName = "app.bsky.feed.post"
	maxPostLength  = 300
)

type Bluesky interface {
	Post(ctx context.Context, handle string, item rss.NewsItem) error
}

type blueskyImpl struct {
	client *xrpc.Client
}

func NewBluesky(ctx context.Context, identifier, password string) (Bluesky, error) {
	client := &xrpc.Client{Host: defaultPDSHost}

	out, err := atproto.ServerCreateSession(ctx, client, &atproto.ServerCreateSession_Input{
		Identifier: identifier,
		Password:   password,
	})
	if err != nil {
		return nil, err
	}

	client.Auth = &xrpc.AuthInfo{
		AccessJwt:  out.AccessJwt,
		RefreshJwt: out.RefreshJwt,
		Did:        out.Did,
		Handle:     out.Handle,
	}

	return &blueskyImpl{client: client}, nil
}

func (b *blueskyImpl) Post(ctx context.Context, handle string, item rss.NewsItem) error {
	when := time.Now().Format(time.RFC3339)
	postText := constructPostText(item)
	tags := generateTags(item.Categories)
	facets := constructFacets(postText, tags)

	post := &bsky.FeedPost{
		LexiconTypeID: collectionName,
		Text:          postText,
		CreatedAt:     when,
		Facets:        facets,
		Embed: &bsky.FeedPost_Embed{
			EmbedExternal: &bsky.EmbedExternal{
				LexiconTypeID: "app.bsky.embed.external",
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
	return err
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
						LexiconTypeID: "app.bsky.richtext.facet#tag",
						Tag:           tag[1:], // Remove the '#' from the tag
					},
				},
			},
		})
	}
	return facets
}