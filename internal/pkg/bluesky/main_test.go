package bluesky

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/thulasirajkomminar/aws-news-bot/internal/pkg/rss"
)

func TestBuildPost_FitsWithinBlueskyLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		item *rss.NewsItem
		tags []string
	}{
		{
			name: "short title and description",
			item: &rss.NewsItem{
				Title:       "Short title",
				Description: "Short description.",
			},
			tags: []string{"#AWS"},
		},
		{
			name: "long title forces title truncation",
			item: &rss.NewsItem{
				Title:       strings.Repeat("A", 350),
				Description: "irrelevant",
			},
			tags: []string{"#AWS", "#Lambda"},
		},
		{
			name: "long description forces description truncation",
			item: &rss.NewsItem{
				Title:       "Brief",
				Description: strings.Repeat("d", 500),
			},
			tags: []string{"#AWS"},
		},
		{
			name: "non-ASCII characters counted by rune",
			item: &rss.NewsItem{
				Title:       strings.Repeat("é", 200),
				Description: strings.Repeat("ü", 200),
			},
			tags: []string{"#AWS"},
		},
		{
			name: "many tags shrink available body",
			item: &rss.NewsItem{
				Title:       strings.Repeat("T", 100),
				Description: strings.Repeat("D", 200),
			},
			tags: []string{"#AWS", "#Lambda", "#DynamoDB", "#S3", "#EC2", "#RDS"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			text, _ := buildPost(tt.item, tt.tags)
			runeCount := utf8.RuneCountInString(text)

			if runeCount > maxPostLength {
				t.Fatalf("post exceeds %d runes: got %d for text %q", maxPostLength, runeCount, text)
			}
		})
	}
}

func TestBuildPost_FacetsAnchorToTagBlock(t *testing.T) {
	t.Parallel()

	item := &rss.NewsItem{
		Title:       "AWS Lambda update #AWS in title",
		Description: "Pre-existing #AWS reference inside the description body.",
	}
	tags := []string{"#AWS"}

	text, facets := buildPost(item, tags)

	if len(facets) != 1 {
		t.Fatalf("expected 1 facet, got %d", len(facets))
	}

	start := facets[0].Index.ByteStart
	end := facets[0].Index.ByteEnd

	if got := text[start:end]; got != "#AWS" {
		t.Fatalf("facet bytes %d-%d are %q, want %q", start, end, got, "#AWS")
	}

	firstHash := strings.Index(text, "#AWS")
	if int64(firstHash) == start {
		t.Fatalf("facet anchored at inline occurrence (%d), want trailing block", firstHash)
	}
}

func TestTruncateRunes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		budget int
		want   string
	}{
		{"fits exactly", "hello", 5, "hello"},
		{"fits with room", "hi", 5, "hi"},
		{"truncated", "hello world", 8, "hello..."},
		{"budget below suffix length", "hello", 2, ".."},
		{"zero budget", "hello", 0, ""},
		{"negative budget", "hello", -1, ""},
		{"unicode preserved", "héllo wörld", 8, "héllo..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := truncateRunes(tt.input, tt.budget, "...")
			if got != tt.want {
				t.Fatalf("truncateRunes(%q, %d) = %q, want %q", tt.input, tt.budget, got, tt.want)
			}
		})
	}
}

func TestCategoryToTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input  string
		want   string
		wantOK bool
	}{
		{"general:products/aws-lambda", "#AwsLambda", true},
		{"general:products/amazon-s3", "#AmazonS3", true},
		{"news", "#News", true},
		{"big-launch", "#BigLaunch", true},
		{"general:announcements", "", false},
		{"foo/bar", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			got, ok := categoryToTag(tt.input)
			if ok != tt.wantOK || got != tt.want {
				t.Fatalf("categoryToTag(%q) = (%q, %v), want (%q, %v)", tt.input, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestRkeyFor_StableAndValid(t *testing.T) {
	t.Parallel()

	first := rkeyFor("https://example.com/post-123")
	again := rkeyFor("https://example.com/post-123")
	other := rkeyFor("https://example.com/post-124")

	if first != again {
		t.Fatalf("rkeyFor not deterministic: %q vs %q", first, again)
	}

	if first == other {
		t.Fatalf("rkeyFor produced collision for distinct keys")
	}

	if got, want := len(first), tidLength; got != want {
		t.Fatalf("rkey length = %d, want %d", got, want)
	}

	for i, r := range first {
		if !strings.ContainsRune(tidAlphabet, r) {
			t.Fatalf("rkey %q contains rune %q outside TID alphabet", first, r)
		}

		// First char must lie in the first half so the TID's top bit is zero.
		if i == 0 && !strings.ContainsRune(tidAlphabet[:16], r) {
			t.Fatalf("rkey %q has invalid first char %q (top bit must be 0)", first, r)
		}
	}
}

func TestIsDuplicateRecordErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		want bool
	}{
		{errAlreadyExists("Record already exists in repo"), true},
		{errAlreadyExists("InvalidRequest: RecordAlreadyExists"), true},
		{errAlreadyExists("duplicate record"), true},
		{errAlreadyExists("network timeout"), false},
		{errAlreadyExists("unauthorized"), false},
	}

	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			t.Parallel()

			if got := isDuplicateRecordErr(tt.err); got != tt.want {
				t.Fatalf("isDuplicateRecordErr(%q) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

type stringError string

func (e stringError) Error() string { return string(e) }

func errAlreadyExists(s string) error { return stringError(s) }

func TestGenerateTags_Deduplicates(t *testing.T) {
	t.Parallel()

	categories := []string{
		"general:products/aws-lambda",
		"aws-lambda",
		"general:products/aws-lambda",
	}

	tags := generateTags(categories)

	seen := make(map[string]int, len(tags))
	for _, tag := range tags {
		seen[tag]++
	}

	for tag, count := range seen {
		if count > 1 {
			t.Fatalf("tag %q appeared %d times: %v", tag, count, tags)
		}
	}
}
