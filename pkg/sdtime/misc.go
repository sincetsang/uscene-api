package sdtime

import (
	"time"
)

func NowTruncateSecond() time.Time {
	return time.Now().Truncate(time.Second)
}
