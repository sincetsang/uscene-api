package sdredis

import (
	"fmt"
	"github.com/go-redis/redis/v8"

	"TMA/pkg/sderr"
)

type Client struct {
	redis.UniversalClient
}

var _ redis.Cmdable = (*Client)(nil)

func (c *Client) String() string {
	if c.UniversalClient == nil {
		return "Nil"
	}
	c1, ok := c.UniversalClient.(fmt.Stringer)
	if !ok {
		return "Redis"
	}
	return c1.String()
}

func (c *Client) Close() error {
	type closeable interface {
		Close() error
	}
	if c.UniversalClient == nil {
		return sderr.New("nil cmdable")
	}
	closable, ok := c.UniversalClient.(closeable)
	if !ok {
		return sderr.New("not closable")
	}
	return closable.Close()
}
