package sdecho

type PagingOptions struct {
	NumArg, SizeArg string
	DefaultSize     int
	BaseZero        bool
}

func (po *PagingOptions) get() (string, string) {
	if po == nil {
		return "page", "size"
	}
	numArg, sizeArg := po.NumArg, po.SizeArg
	if numArg == "" {
		numArg = "page"
	}
	if sizeArg == "" {
		sizeArg = "size"
	}
	return numArg, sizeArg
}

func (po *PagingOptions) trimNum(n int) int {
	baseZero := false
	if po != nil {
		baseZero = po.BaseZero
	}
	if baseZero {
		if n < 0 {
			return 0
		}
		return n
	} else {
		if n < 1 {
			return 1
		}
		return n
	}
}

func (po *PagingOptions) defaultSize() int {
	if po == nil {
		return 10
	}
	defaultSize := po.DefaultSize
	if defaultSize <= 0 {
		return 10
	}
	return defaultSize
}

func (ec Context) PageNum(opt *PagingOptions) int {
	numArg, _ := opt.get()
	return opt.trimNum(ec.ArgInt(numArg, 0))
}

func (ec Context) PageSize(opt *PagingOptions) int {
	_, sizeArg := opt.get()
	pageSize := ec.ArgInt(sizeArg, 0)
	if pageSize <= 0 {
		return opt.defaultSize()
	}
	return pageSize
}

func (ec Context) PageSizeFixed(opt *PagingOptions) int {
	return opt.defaultSize()
}
