package remind

import "time"

// Each weekday's slots, in local hours.
const (
	morning   = 10
	afternoon = 15
)

// nextSlot returns the first slot after from: 10:00 or 15:00, Monday to Friday, in location.
func nextSlot(from time.Time, location *time.Location) time.Time {
	year, month, day := from.In(location).Date()

	for days := 0; ; days++ {
		for _, hour := range [...]int{morning, afternoon} {
			slot := time.Date(year, month, day+days, hour, 0, 0, 0, location)
			if slot.After(from) && isWeekday(slot) {
				return slot
			}
		}
	}
}

// isWeekday reports whether t falls on Monday to Friday, in its own location.
func isWeekday(t time.Time) bool {
	return t.Weekday() != time.Saturday && t.Weekday() != time.Sunday
}

// later returns the later of a and b.
func later(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}

	return b
}
