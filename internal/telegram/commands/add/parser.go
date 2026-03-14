package add

import (
	"strconv"
	"strings"
)

func parseAddTaskNumber(text string) (int64, bool) {
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) != 2 {
		return 0, false
	}

	taskNumber, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || taskNumber <= 0 {
		return 0, false
	}

	return taskNumber, true
}
