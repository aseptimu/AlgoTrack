package timezone

import "time"

// MoscowLocation is the pre-loaded Europe/Moscow timezone.
// Loaded once at package init to avoid repeated disk reads from time.LoadLocation.
var MoscowLocation *time.Location

func init() {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		loc = time.FixedZone("MSK", 3*60*60)
	}
	MoscowLocation = loc
}
