package sdrand

import (
	"math/rand"
	"reflect"

	"TMA/pkg/sderr"
)

type W struct {
	W int         `json:"w"`
	V interface{} `json:"v"`
}

func ChoiceOneStr(choices ...string) string {
	n := len(choices)
	if n == 0 {
		return ""
	}
	return choices[rand.Intn(n)]
}

func ChoiceOneInt(choices ...int) int {
	n := len(choices)
	if n == 0 {
		return 0
	}
	return choices[rand.Intn(n)]
}

func ChoiceOneInt64(choices ...int64) int64 {
	n := len(choices)
	if n == 0 {
		return 0
	}
	return choices[rand.Intn(n)]
}

func ChoiceOne(choicesArray interface{}) interface{} {
	if choicesArray == 0 {
		return nil
	}
	v := reflect.ValueOf(choicesArray)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		panic(sderr.WithStack(sderr.ErrIllegalType))
	}

	n := v.Len()
	if n == 0 {
		return nil
	}
	return v.Index(rand.Intn(n)).Interface()
}

func ChoiceWeighted(choices ...W) interface{} {
	n := len(choices)
	if n == 0 {
		return nil
	}
	if n == 1 {
		first := choices[0]
		if first.W > 0 {
			return first.V
		} else {
			return nil
		}
	}
	var sum, upto int64 = 0, 0
	for _, w := range choices {
		if w.W > 0 {
			sum += int64(w.W)
		}
	}
	r := BetweenFloat64(0.0, float64(sum))
	for _, w := range choices {
		ww := w.W
		if ww < 0 {
			ww = 0
		}
		if float64(upto)+float64(ww) > r {
			return w.V
		}
		upto += int64(w.W)
	}
	return nil
}

func ChoiceSomeStr(choices []string, n int) []string {
	nChoice := len(choices)
	if nChoice == 0 || n <= 0 {
		return []string{}
	}
	m := make(map[int]string, nChoice)
	for i, v := range choices {
		m[i] = v
	}

	r := make([]string, 0, n)
	for i := 0; i < n; i++ {
		if len(m) <= 0 || len(r) >= n {
			break
		}
		c := rand.Intn(len(m))
		j := 0
	Next:
		for arrIndex, v := range m {
			if j == c {
				r = append(r, v)
				delete(m, arrIndex)
				break Next
			}
			j++
		}
	}
	return r
}

// TODO: ChoiceSomeInt
// TODO: ChoiceSomeInt64

func ChoiceSome(choicesArray interface{}, n int) interface{} {
	v := reflect.ValueOf(choicesArray)
	k := v.Kind()
	if k != reflect.Slice && k != reflect.Array {
		panic(sderr.WithStack(sderr.ErrIllegalType))
	}

	nChoice := v.Len()
	if nChoice == 0 || n <= 0 {
		return reflect.MakeSlice(reflect.SliceOf(v.Type().Elem()), 0, 0).Interface()
	}
	m := make(map[int]reflect.Value, nChoice)
	for i := 0; i < nChoice; i++ {
		m[i] = v.Index(i)
	}

	r := reflect.MakeSlice(reflect.SliceOf(v.Type().Elem()), 0, n)
	for i := 0; i < n; i++ {
		if len(m) <= 0 || r.Len() >= n {
			break
		}
		c := rand.Intn(len(m))
		j := 0
	Next:
		for arrIndex, v := range m {
			if j == c {
				r = reflect.Append(r, v)
				delete(m, arrIndex)
				break Next
			}
			j++
		}
	}
	return r.Interface()
}
