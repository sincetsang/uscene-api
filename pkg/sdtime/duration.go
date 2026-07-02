package sdtime

import (
	"time"
)

func Millis(n int64) time.Duration {
	return time.Duration(n) * time.Millisecond
}

func Seconds(n int64) time.Duration {
	return time.Duration(n) * time.Second
}

func Minutes(n int64) time.Duration {
	return time.Duration(n) * time.Minute
}

func Hours(n int64) time.Duration {
	return time.Duration(n) * time.Hour
}

func Days(n int64) time.Duration {
	return time.Duration(n*60*24) * time.Minute
}

func ToMillis(d time.Duration) int64 {
	return d.Nanoseconds() / (1000 * 1000)
}

func ToMillisF(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / (1000.0 * 1000.0)
}
