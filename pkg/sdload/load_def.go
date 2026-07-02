package sdload

func StringDef(loc, def string) string {
	s, err := String(loc)
	if err != nil {
		return def
	}
	return s
}
