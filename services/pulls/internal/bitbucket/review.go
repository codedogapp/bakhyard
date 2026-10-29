package bitbucket

// Verdict is where someone who reviews a pull request stands.
type Verdict string

const (
	// Pending is the verdict of a reviewer who has neither approved nor requested changes.
	Pending Verdict = "pending"
	// Approved is the verdict of someone who has approved the pull request.
	Approved Verdict = "approved"
	// ChangesRequested is the verdict of someone who has requested changes.
	ChangesRequested Verdict = "changes_requested"
)

// Reviewer is someone who reviews a pull request: one of its reviewers, or someone else who has
// approved it or requested changes.
type Reviewer struct {
	// Name is the reviewer's display name on Bitbucket.
	Name    string  `json:"name"`
	Verdict Verdict `json:"verdict"`
}

// participant is the part of a pull request's participant in Bitbucket's answer that the client
// reads. A participant is one of its reviewers, or someone else who has commented on it, approved
// it or requested changes.
type participant struct {
	User     account `json:"user"`
	Role     role    `json:"role"`
	Approved bool    `json:"approved"`
	State    state   `json:"state"`
}

// role is a participant's role in Bitbucket's answer: REVIEWER or PARTICIPANT.
type role string

// state is a participant's state in Bitbucket's answer: approved, changes_requested or null.
type state string

const (
	// reviewerRole is the role of one of a pull request's reviewers.
	reviewerRole role = "REVIEWER"
	// changesRequestedState is the state of someone who has requested changes.
	changesRequestedState state = "changes_requested"
)

// account is the part of a Bitbucket account, a user's or an app's, that the client reads.
type account struct {
	UUID        string `json:"uuid"`
	DisplayName string `json:"display_name"`
}

// reviewers returns those among participants who review the pull request, in Bitbucket's order:
// each of its reviewers, and each other participant who has approved it or requested changes.
// It leaves out the pull request's author, whose uuid is author, since Bitbucket lets the author
// approve their own pull request.
func reviewers(participants []participant, author string) []Reviewer {
	var found []Reviewer

	for _, someone := range participants {
		verdict := someone.verdict()
		if someone.User.UUID == author || (someone.Role != reviewerRole && verdict == Pending) {
			continue
		}

		found = append(found, Reviewer{Name: someone.User.DisplayName, Verdict: verdict})
	}

	return found
}

// verdict returns where someone stands. Bitbucket marks an approval in both approved and state,
// and a request for changes in state alone.
func (someone participant) verdict() Verdict {
	switch {
	case someone.Approved:
		return Approved
	case someone.State == changesRequestedState:
		return ChangesRequested
	default:
		return Pending
	}
}
