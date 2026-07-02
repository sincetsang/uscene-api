package sdecho

import (
	"strconv"
	"time"

	"TMA/pkg/sdtime"
)

func (ec Context) ArgStr(name, def string) string {
	v := ec.QueryParam(name)
	if v != "" {
		return v
	}
	v = ec.Param(name)
	if v != "" {
		return v
	}
	return def
}

func (ec Context) ArgInt(name string, def int) int {
	s := ec.ArgStr(name, "")
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func (ec Context) ArgInt64(name string, def int64) int64 {
	s := ec.ArgStr(name, "")
	if s == "" {
		return def
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return v
}

func (ec Context) ArgBool(name string, def bool) bool {
	s := ec.ArgStr(name, "")
	if s == "" {
		return def
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return v
}

func (ec Context) ArgTime(name string, def time.Time) time.Time {
	s := ec.ArgStr(name, "")
	if s == "" {
		return def
	}
	t, err := sdtime.Parse(s)
	if err != nil {
		return def
	}
	return t
}

func (ec Context) FirstArgStr(names []string, def string) string {
	for _, name := range names {
		if name == "" {
			continue
		}
		if arg := ec.ArgStr(name, ""); arg != "" {
			return arg
		}
	}
	return def
}

func (ec Context) FirstArgInt(names []string, def int) int {
	s := ec.FirstArgStr(names, "")
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func (ec Context) FirstArgInt65(names []string, def int64) int64 {
	s := ec.FirstArgStr(names, "")
	if s == "" {
		return def
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return v
}

func (ec Context) FirstArgBool(names []string, def bool) bool {
	s := ec.FirstArgStr(names, "")
	if s == "" {
		return def
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return v
}

func (ec Context) FirstArgTime(names []string, def time.Time) time.Time {
	s := ec.FirstArgStr(names, "")
	if s == "" {
		return def
	}
	v, err := sdtime.Parse(s)
	if err != nil {
		return def
	}
	return v
}
