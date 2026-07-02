package sdtime

import "time"

func Yesterday() time.Time {

	ts := time.Now().AddDate(0, 0, -1)
	return time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, ts.Location())
}
