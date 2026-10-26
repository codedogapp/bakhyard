package todo

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxTitle is the most characters a title can have, counted in Unicode code points (decision
// 106).
const maxTitle = 500

// ErrTitle is wrapped by the error of a title that breaks the rule of decision 106.
var ErrTitle = errors.New("invalid title")

// cleanTitle returns raw without its leading and trailing white space. What remains must have 1
// to maxTitle characters and no control character, or cleanTitle returns an error that wraps
// ErrTitle.
func cleanTitle(raw string) (string, error) {
	title := strings.TrimSpace(raw)

	switch {
	case title == "":
		return "", fmt.Errorf("%w: empty once trimmed", ErrTitle)
	case utf8.RuneCountInString(title) > maxTitle:
		return "", fmt.Errorf("%w: over %d characters", ErrTitle, maxTitle)
	case strings.ContainsFunc(title, unicode.IsControl):
		return "", fmt.Errorf("%w: has a control character", ErrTitle)
	}

	return title, nil
}
