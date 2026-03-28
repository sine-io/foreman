package sqlite

import "time"

const sortableTimestampLayout = "2006-01-02T15:04:05.000000000Z"

func sortableTimestamp(t time.Time) string {
	return t.UTC().Format(sortableTimestampLayout)
}

func normalizeSortableTimestamp(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return "", err
	}

	return sortableTimestamp(parsed), nil
}
