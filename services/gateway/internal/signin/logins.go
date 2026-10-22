package signin

import (
	"maps"
	"sync"
	"time"
)

// loginTTL is how long a login may stay at Google.
const loginTTL = 10 * time.Minute

// pending is a login that went to Google and hasn't come back yet.
type pending struct {
	nonce    string
	verifier string
	expires  time.Time
}

// logins holds the pending logins, by state, in memory.
type logins struct {
	mu      sync.Mutex
	byState map[string]pending
}

func newLogins() *logins {
	return &logins{byState: make(map[string]pending)}
}

// add keeps login under state for loginTTL, and drops the logins that expired.
func (l *logins) add(state string, login pending) {
	now := time.Now()
	login.expires = now.Add(loginTTL)

	l.mu.Lock()
	defer l.mu.Unlock()

	maps.DeleteFunc(l.byState, func(_ string, old pending) bool {
		return !now.Before(old.expires)
	})
	l.byState[state] = login
}

// take removes the login under state and returns it, unless it's unknown or expired.
func (l *logins) take(state string) (pending, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	login, ok := l.byState[state]
	delete(l.byState, state)

	return login, ok && time.Now().Before(login.expires)
}
