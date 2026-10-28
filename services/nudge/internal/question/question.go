// Package question keeps the questions that agents ask, in nudge's database, and hands each
// answer to the agent that asked.
package question

import (
	"errors"
	"slices"
	"strings"
	"time"
	"uuid"
)

var (
	// ErrBadID means the question's ID isn't a UUID.
	ErrBadID = errors.New("the question's id isn't a UUID")
	// ErrNoText means the question has no text.
	ErrNoText = errors.New("the question has no text")
	// ErrBadOptions means one of the question's options is empty or repeated.
	ErrBadOptions = errors.New("an option is empty or repeated")
	// ErrEmptyAnswer means the answer has neither an option nor a text.
	ErrEmptyAnswer = errors.New("the answer has neither an option nor a text")
	// ErrUnknownOption means the answer picks an option that the question doesn't offer.
	ErrUnknownOption = errors.New("the question doesn't offer the answer's option")
	// ErrNotFound means the question has been answered already, cancelled or has expired.
	ErrNotFound = errors.New("no such question")
)

const (
	// pendingTTL is how long a question waits for an answer before it expires.
	pendingTTL = 24 * time.Hour
	// answeredTTL is how long an answered question stays, so that an agent whose connection broke
	// can still fetch its answer.
	answeredTTL = 7 * 24 * time.Hour
)

// Question is what an agent asks. The agent gives it its ID, so that it can ask again after a
// broken connection and get the same question.
type Question struct {
	ID      string    `json:"id"`
	Project string    `json:"project"`
	Text    string    `json:"text"`
	Details string    `json:"details"`
	Options []string  `json:"options"`
	AskedAt time.Time `json:"asked_at"`
}

// Answer is the user's reply: one of the question's options, a text, or both.
type Answer struct {
	Option string `json:"option"`
	Text   string `json:"text"`
}

// check reports why the question can't be asked, if it can't.
func (q Question) check() error {
	if _, err := uuid.Parse(q.ID); err != nil {
		return ErrBadID
	}

	if strings.TrimSpace(q.Text) == "" {
		return ErrNoText
	}

	for i, option := range q.Options {
		if strings.TrimSpace(option) == "" || slices.Contains(q.Options[:i], option) {
			return ErrBadOptions
		}
	}

	return nil
}

// accepts reports why the answer doesn't fit the question, if it doesn't.
func (q Question) accepts(answer Answer) error {
	if answer.Option == "" && answer.Text == "" {
		return ErrEmptyAnswer
	}

	if answer.Option != "" && !slices.Contains(q.Options, answer.Option) {
		return ErrUnknownOption
	}

	return nil
}
