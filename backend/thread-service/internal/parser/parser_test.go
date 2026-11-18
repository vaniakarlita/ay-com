package parser_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vaniakarlita/ay-com/backend/thread-service/internal/parser"
)

func TestExtractHashtags(t *testing.T) {
	text := "Hello #world! Welcome to #AY_com and #golang. Duplicate #world."
	tags := parser.ExtractHashtags(text)

	assert.Len(t, tags, 3)
	assert.Contains(t, tags, "world")
	assert.Contains(t, tags, "ay_com")
	assert.Contains(t, tags, "golang")
}

func TestExtractMentions(t *testing.T) {
	text := "Hi @adrian and @john_doe, check this with @adrian."
	mentions := parser.ExtractMentions(text)

	assert.Len(t, mentions, 2)
	assert.Contains(t, mentions, "adrian")
	assert.Contains(t, mentions, "john_doe")
}

func TestNoTagsOrMentions(t *testing.T) {
	text := "Just a normal post without anything special."
	assert.Nil(t, parser.ExtractHashtags(text))
	assert.Nil(t, parser.ExtractMentions(text))
}
