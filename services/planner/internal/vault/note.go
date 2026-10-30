package vault

import (
	"cmp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// folder is the vault's folder of plans.
const folder = "Plans"

// untitled is the title of a plan that doesn't start with an H1.
const untitled = "Plan"

// maxSlug is the most runes that a note's name takes from its plan's title, which keeps the name
// under the 255 bytes that notes takes.
const maxSlug = 60

// note is a plan as a note of the vault.
type note struct {
	path string
	text string
}

// newNote returns plan as a note made on day: in the folder Plans, named with the day and the
// plan's title, with the frontmatter that the vault's notes have, a blank line, then the plan.
func newNote(plan string, day time.Time) note {
	title := titleOf(plan)
	date := day.Format(time.DateOnly)
	// strconv.Quote's escapes are all YAML's too, so the title stays a valid double-quoted scalar.
	front := "---\ntitle: " + strconv.Quote(title) + "\ntags:\n  - \"plan\"\ncreated: " + date +
		"\n---\n\n"

	return note{path: folder + "/" + date + "-" + slug(title) + ".md", text: front + plan}
}

// titleOf returns the text of the H1 that plan starts with, or untitled.
func titleOf(plan string) string {
	first, _, _ := strings.Cut(strings.TrimLeft(plan, " \t\r\n"), "\n")
	text, found := strings.CutPrefix(first, "# ")

	text = strings.TrimSpace(text)
	if !found || text == "" {
		return untitled
	}

	return text
}

// slug returns title in lower case, without apostrophes, with its words, the runs of letters,
// marks and digits, joined by hyphens: as many whole words as maxSlug runes take, or the first
// word's first maxSlug runes. A title without a word gives "plan".
func slug(title string) string {
	plain := strings.NewReplacer("'", "", "\u2019", "").Replace(strings.ToLower(title))
	name := ""

	for word := range strings.FieldsFuncSeq(plain, notInWord) {
		next := strings.TrimPrefix(name+"-"+word, "-")
		if utf8.RuneCountInString(next) <= maxSlug {
			name = next

			continue
		}

		if name == "" {
			name = string([]rune(word)[:maxSlug])
		}

		break
	}

	return cmp.Or(name, "plan")
}

// notInWord reports whether r separates the words of a title.
func notInWord(r rune) bool {
	return !unicode.In(r, unicode.L, unicode.M, unicode.Nd)
}
