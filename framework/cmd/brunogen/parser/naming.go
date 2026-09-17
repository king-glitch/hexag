package parser

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	matchFirstCap = regexp.MustCompile("([a-z0-9])([A-Z])")
	matchAllCap   = regexp.MustCompile("([A-Z]+)([A-Z][a-z0-9])")
)

// ToSnakeCase mirrors cmd/mongogen's helper of the same name, used as the
// fallback JSON/query key when a field has no tag.
func ToSnakeCase(str string) string {
	snake := matchAllCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchFirstCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// ToKebabCase mirrors cmd/mongogen's helper of the same name, used for
// generated file/folder names.
func ToKebabCase(str string) string {
	var result []rune
	runes := []rune(str)

	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			var next rune
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			if unicode.IsLower(prev) || (unicode.IsUpper(prev) && next != 0 && unicode.IsLower(next)) {
				result = append(result, '-')
			}
		}
		result = append(result, unicode.ToLower(r))
	}

	return string(result)
}
