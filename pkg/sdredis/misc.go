package sdredis

import (
	"TMA/pkg/sderr"
	"github.com/go-redis/redis/v8"
)

func process(c redis.Cmdable, cmd redis.Cmder) error {
	type processable interface {
		Process(cmd redis.Cmder) error
	}

	if p1, ok := c.(processable); ok {
		return p1.Process(cmd)
	} else {
		return sderr.New("missing 'process'")
	}
}
