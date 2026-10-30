package exophase

import (
	"math"
	"time"
)

// Game is a game that the player played, as the games service answers it.
type Game struct {
	// Title is the game's name on Exophase.
	Title string `json:"title"`
	// Platforms are the platforms that Exophase lists for the game, such as PS5 or Switch.
	Platforms []string `json:"platforms"`
	// LastPlayed is when the player last played the game.
	LastPlayed time.Time `json:"lastPlayed"`
	// PlaytimeMinutes is how long the player has played the game, or 0 when Exophase doesn't know.
	PlaytimeMinutes int `json:"playtimeMinutes,omitzero"`
	// Trophies are zero for a game without trophies or achievements, such as Nintendo's.
	Trophies Trophies `json:"trophies,omitzero"`
}

// Trophies count the trophies or achievements that the player earned in a game.
type Trophies struct {
	Earned int `json:"earned"`
	Total  int `json:"total"`
	// Percent is how far the player got through them, as Exophase reckons it.
	Percent float64 `json:"percent"`
}

// listPage is the part of a page of the player's games list that the client reads.
type listPage struct {
	Success bool         `json:"success"`
	Games   []listedGame `json:"games"`
}

// listedGame is the part of a game in the list that the client reads.
type listedGame struct {
	// LastPlayed is in Unix seconds.
	LastPlayed int64    `json:"lastplayed_utc"`
	Earned     int      `json:"earned_awards"`
	Total      int      `json:"total_awards"`
	Percent    float64  `json:"percent"`
	Playtime   playtime `json:"playtimeUnits"`
	Meta       meta     `json:"meta"`
}

// playtime is a game's playtime, which is 0 and 0 when Exophase doesn't know it.
type playtime struct {
	Hours   float64 `json:"hours"`
	Minutes float64 `json:"minutes"`
}

// meta is the part of a game's details that the client reads.
type meta struct {
	Title     string     `json:"title"`
	Platforms []platform `json:"platforms"`
}

// platform is a platform that Exophase lists for a game.
type platform struct {
	Name string `json:"name"`
}

// game returns the Game of a game in the list.
func (g listedGame) game() Game {
	platforms := make([]string, 0, len(g.Meta.Platforms))
	for _, listed := range g.Meta.Platforms {
		platforms = append(platforms, listed.Name)
	}

	game := Game{
		Title:           g.Meta.Title,
		Platforms:       platforms,
		LastPlayed:      time.Unix(g.LastPlayed, 0).UTC(),
		PlaytimeMinutes: max(0, int(math.Round(g.Playtime.Hours*60+g.Playtime.Minutes))),
	}
	if g.Total > 0 {
		game.Trophies = Trophies{Earned: g.Earned, Total: g.Total, Percent: g.Percent}
	}

	return game
}
