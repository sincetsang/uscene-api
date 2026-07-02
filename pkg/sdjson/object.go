package sdjson

type Object map[string]interface{}

func (o Object) Len() int {
	return len(o)
}

func (o Object) Has(k string) bool {
	_, ok := o[k]
	return ok
}

func (o Object) Get(k string) Value {
	v0, ok := o[k]
	if ok {
		return V(v0)
	} else {
		return V(nil)
	}
}

func (o Object) Set(k string, v interface{}) Object {
	if o != nil {
		o[k] = unbox(v)
	}
	return o
}
