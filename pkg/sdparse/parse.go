package sdparse

import (
	"strconv"
	"time"

	"TMA/pkg/sderr"
)

func Int64(s string) (int64, error) {
	r, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, sderr.WithStack(err)
	}
	return r, nil
}

func Int(s string) (int, error) {
	r, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, sderr.WithStack(err)
	}
	return int(r), nil
}

func Uint64(s string) (uint64, error) {
	r, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, sderr.WithStack(err)
	}
	return r, nil
}

func Uint(s string) (uint, error) {
	r, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, sderr.WithStack(err)
	}
	return uint(r), nil
}

func Float64(s string) (float64, error) {
	r, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0, sderr.WithStack(err)
	}
	return r, nil
}

func Bool(s string) (bool, error) {
	r, err := strconv.ParseBool(s)
	if err != nil {
		return false, sderr.WithStack(err)
	}
	return r, nil
}

var (
	timeLayoutsForParse = []string{
		"20060102150405",
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.Kitchen,
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02",                         // RFC 3339
		"2006-01-02 15:04",                   // RFC 3339 with minutes
		"2006-01-02 15:04:05",                // RFC 3339 with seconds
		"2006-01-02 15:04:05-07:00",          // RFC 3339 with seconds and timezone
		"2006-01-02T15Z0700",                 // ISO8601 with hour
		"2006-01-02T15:04Z0700",              // ISO8601 with minutes
		"2006-01-02T15:04:05Z0700",           // ISO8601 with seconds
		"2006-01-02T15:04:05.999999999Z0700", // ISO8601 with nanoseconds
	}
)

func Time(s string) (time.Time, error) {
	for _, layout := range timeLayoutsForParse {
		r, err := time.Parse(layout, s)
		if err == nil {
			return r, nil
		}
	}
	return time.Time{}, sderr.New("parse time error")
}
