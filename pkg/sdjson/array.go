package sdjson

type Array []interface{}

func (a Array) Len() int {
	return len(a)
}

func (a Array) Has(i int) bool {
	return 0 <= i && i < len(a)
}

func (a Array) At(i int) Value {
	if 0 <= i && i < len(a) {
		return V(a[i])
	} else {
		return V(nil)
	}
}
