package sqlite

import "time"

const sortableTimestampLayout = "2006-01-02T15:04:05.000000000Z"

func sortableTimestamp(t time.Time) string {
	return t.UTC().Format(sortableTimestampLayout)
}
