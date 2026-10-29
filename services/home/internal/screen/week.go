package screen

// Week is the hours logged this week, against the weekly target, as the time service answers it.
type Week struct {
	LoggedHours float64 `json:"loggedHours"`
	TargetHours float64 `json:"targetHours"`
}
