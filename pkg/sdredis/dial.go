package sdredis

import (
	"github.com/go-redis/redis/v8"
	"time"

	"TMA/pkg/sderr"
)

type Address struct {
	Addrs      []string `json:"addrs" toml:"addrs"`
	Database   int      `json:"db,omitempty" toml:"db"`
	Password   string   `json:"password,omitempty" toml:"password"`
	MaxRetries int      `json:"max_retries,omitempty" toml:"max_retries"`
	Cluster    bool     `json:"cluser,omitempty" toml:"cluster"`
}

func Dial(addr Address) (*Client, error) {
	const (
		defaultPoolSize    = 30
		defaultPoolTimeout = 60 * time.Second
	)

	switch len(addr.Addrs) {
	case 0:
		return nil, sderr.New("no addrs")
	case 1:
		client := redis.NewClient(&redis.Options{
			Addr:        addr.Addrs[0],
			Password:    addr.Password,
			DB:          addr.Database,
			MaxRetries:  addr.MaxRetries,
			PoolSize:    defaultPoolSize,
			PoolTimeout: defaultPoolTimeout,
		})
		return &Client{UniversalClient: client}, nil
	default:
		if addr.Cluster {
			client := redis.NewClusterClient(&redis.ClusterOptions{
				Addrs:       addr.Addrs,
				Password:    addr.Password,
				MaxRetries:  addr.MaxRetries,
				PoolSize:    defaultPoolSize,
				PoolTimeout: defaultPoolTimeout,
			})
			return &Client{UniversalClient: client}, nil
		} else {
			addrMap := map[string]string{}
			for _, addr1 := range addr.Addrs {
				addrMap[addr1] = addr1
			}
			client := redis.NewRing(&redis.RingOptions{
				Addrs:       addrMap,
				Password:    addr.Password,
				DB:          addr.Database,
				MaxRetries:  addr.MaxRetries,
				PoolSize:    defaultPoolSize,
				PoolTimeout: defaultPoolTimeout,
			})
			return &Client{UniversalClient: client}, nil
		}
	}
}
