package app

import "time"

// nowUTC returns the current time in UTC. Extracted as a function to allow
// simple deterministic testing of buildRecord without needing time injection.
func nowUTC() time.Time {
	return time.Now().UTC()
}
