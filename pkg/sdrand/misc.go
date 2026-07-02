package sdrand

import (
	"math/rand"
	"time"
)

func InitSeed() {
	rand.Seed(time.Now().UnixNano())
}
