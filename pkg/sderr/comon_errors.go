package sderr

var (
	ErrNilArg      = Sentinel("nil argument")
	ErrIllegalType = Sentinel("illegal type")
)
