package parser

import (
	"regexp"
	"strings"
)

var (
	hashtagRegex = regexp.MustCompile(`#([A-Za-z0-9_]+)`)
	mentionRegex = regexp.MustCompile(`@([A-Za-z0-9_]+)`)
)

func ExtractHashtags(text string) []string {
	matches := hashtagRegex.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var tags []string
	for _, m := range matches {
		if len(m) > 1 {
			tag := strings.ToLower(m[1])
			if !seen[tag] {
				seen[tag] = true
				tags = append(tags, tag)
			}
		}
	}
	return tags
}

func ExtractMentions(text string) []string {
	matches := mentionRegex.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	var usernames []string
	for _, m := range matches {
		if len(m) > 1 {
			u := strings.ToLower(m[1])
			if !seen[u] {
				seen[u] = true
				usernames = append(usernames, u)
			}
		}
	}
	return usernames
}
