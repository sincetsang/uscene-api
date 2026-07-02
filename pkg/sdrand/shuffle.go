package sdrand

import (
	"math"
	"math/rand"
	"reflect"

	"TMA/pkg/sderr"
)

func Shuffle(arr interface{}) {
	typ := reflect.TypeOf(arr)
	if typ.Kind() != reflect.Slice && typ.Kind() != reflect.Array {
		panic(sderr.WithStack(sderr.ErrIllegalType))
	}

	arrVar := reflect.ValueOf(arr)
	n := arrVar.Len()
	if n <= 1 {
		return
	}
	clone := make([]interface{}, n)
	for i := 0; i < n; i++ {
		clone[i] = arrVar.Index(i).Interface()
	}
	perms := rand.Perm(n)
	for i := 0; i < n; i++ {
		newIndex := perms[i]
		v1 := clone[newIndex]
		arrVar.Index(i).Set(reflect.ValueOf(v1))
	}
	return
}

func TruncateToTwoDecimals(value float64) float64 {
	return math.Floor(value*100) / 100
}
