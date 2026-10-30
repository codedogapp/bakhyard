package copilot

import (
	"context"
	"errors"
	"slices"
	"uuid"

	sdk "github.com/github/copilot-sdk/go"
)

var (
	// ErrNoQuestion means that no question with the ID that an answer names waits: it has had
	// its answer already, or the plan never asked it.
	ErrNoQuestion = errors.New("no such question")
	// ErrNotAChoice means that an answer isn't one of the choices of a question that takes no
	// other answer.
	ErrNotAChoice = errors.New("the answer is not one of the choices")
	// errStopped is the CLI's answer to a question that waits as the client stops.
	errStopped = errors.New("the planner stopped")
)

// Question is a question of the agent's, which waits for the user's answer.
type Question struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	// Choices are the answers that the agent offers, if any.
	Choices []string `json:"choices"`
	// Freeform reports whether the question takes an answer other than the choices.
	Freeform bool `json:"freeform"`
}

// Answer is the user's answer to a question.
type Answer struct {
	// Question is the ID of the question that it answers.
	Question string
	Text     string
}

// waiting is a question that waits for its answer, which comes on answers.
type waiting struct {
	Question

	answers chan string
}

// Answer hands answer to its question of the plan id, which then no longer waits. It returns
// ErrNoPlan when the CLI keeps no such plan, ErrNoQuestion when the question doesn't wait, and
// ErrNotAChoice when the answer isn't one of the choices of a question that takes no other.
func (c *Client) Answer(ctx context.Context, id string, answer Answer) error {
	planned, err := c.find(ctx, id)
	if err != nil {
		return err
	}

	return planned.answer(answer)
}

// ask keeps the agent's question and tells the followers, then waits for its answer, which goes
// to the agent, or for the client to stop. The SDK calls it in a goroutine of its own.
func (p *plan) ask(
	request sdk.UserInputRequest,
	_ sdk.UserInputInvocation,
) (sdk.UserInputResponse, error) {
	asked := p.wait(questionOf(request))

	select {
	case answer := <-asked.answers:
		freeform := !slices.Contains(asked.Choices, answer)

		return sdk.UserInputResponse{Answer: answer, WasFreeform: freeform}, nil
	case <-p.stopped:
		return sdk.UserInputResponse{}, errStopped
	}
}

// wait keeps question among those that wait, and tells the followers.
func (p *plan) wait(question Question) waiting {
	asked := waiting{Question: question, answers: make(chan string, 1)}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.questions = append(p.questions, asked)
	p.tell(Update{Change: ChangeQuestion, Question: question})

	return asked
}

// answer hands answer to its question, which then no longer waits, and tells the followers.
func (p *plan) answer(answer Answer) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	index := slices.IndexFunc(p.questions, func(asked waiting) bool {
		return asked.ID == answer.Question
	})
	if index < 0 {
		return ErrNoQuestion
	}

	asked := p.questions[index]
	if !asked.Freeform && !slices.Contains(asked.Choices, answer.Text) {
		return ErrNotAChoice
	}

	p.questions = slices.Delete(p.questions, index, index+1)
	p.tell(Update{Change: ChangeAnswered, Question: asked.Question})

	asked.answers <- answer.Text

	return nil
}

// asking returns the questions that wait. p.mu is held.
func (p *plan) asking() []Question {
	var list []Question

	for _, asked := range p.questions {
		list = append(list, asked.Question)
	}

	return list
}

// questionOf returns the question that request brings, with an ID of its own, since the SDK
// gives none. It takes any answer unless the agent says otherwise, as the CLI's own prompt does,
// and always when it has no choices, which would leave it no answer at all.
func questionOf(request sdk.UserInputRequest) Question {
	freeform := request.AllowFreeform == nil || *request.AllowFreeform || len(request.Choices) == 0

	return Question{
		ID:       uuid.New().String(),
		Text:     request.Question,
		Choices:  request.Choices,
		Freeform: freeform,
	}
}
