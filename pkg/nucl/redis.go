package nucl

import (
	"TMA/pkg/sdredis"
	"github.com/go-redis/redis_rate/v9"
)

func (nu *Nucleus) initRedis() error {
	if !nu.Options.Redis {
		return nil
	}

	return nu.withLog("Redis", func() error {
		client, err := sdredis.Dial(nu.Config.Redis)
		if err != nil {
			return err
		}

		nu.RedisClient = client
		nu.RedisRateLimiter = redis_rate.NewLimiter(client)

		return nil
	})
}
