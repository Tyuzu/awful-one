package hashtags

import (
	"regexp"
	"strings"
)

// ExtractHashtagsFromContent finds hashtags in post text.
// Example: "Loving #golang and #OpenSource" -> ["golang", "opensource"]
func ExtractHashtagsFromContent(content string) []string {
	re := regexp.MustCompile(`#(\w+)`)
	matches := re.FindAllStringSubmatch(content, -1)

	seen := make(map[string]struct{})
	var tags []string

	for _, m := range matches {
		if len(m) > 1 {
			tag := strings.ToLower(m[1]) // normalize to lowercase
			if _, exists := seen[tag]; !exists {
				seen[tag] = struct{}{}
				tags = append(tags, tag)
			}
		}
	}
	return tags
}

/*
// Example Usage
content := "Just released a new project in #GoLang! #OpenSource #golang"
hashtags := ExtractHashtagsFromContent(content)
// hashtags == []string{"golang", "opensource"}

*/
