package sdredis

import (
	"context"
	"github.com/go-redis/redis/v8"

	"TMA/pkg/sderr"
)

type Contextable interface {
	WithContext(ctx context.Context) redis.Cmdable
}

func (c *Client) WithContext(ctx context.Context) redis.Cmdable {
	return WithContext(ctx, c.UniversalClient)
}

func (c *Client) WithContextOpt(ctxOpt context.Context) redis.Cmdable {
	return WithContextOpt(ctxOpt, c.UniversalClient)
}

func WithContext(ctx context.Context, cmdable redis.Cmdable) redis.Cmdable {
	if ctx == nil {
		panic(sderr.New("nil context"))
	}
	return withContext(ctx, cmdable)
}

func WithContextOpt(ctxOpt context.Context, cmdable redis.Cmdable) redis.Cmdable {
	if ctxOpt == nil {
		return cmdable
	}
	return withContext(ctxOpt, cmdable)
}

func withContext(ctx context.Context, cmdable redis.Cmdable) redis.Cmdable {
	type clientContextable interface {
		WithContext(ctx context.Context) *redis.Client
	}
	type ringContextable interface {
		WithContext(ctx context.Context) *redis.Ring
	}
	type clusterContextable interface {
		WithContext(ctx context.Context) *redis.ClusterClient
	}
	if c1, ok := cmdable.(Contextable); ok {
		return c1.WithContext(ctx)
	} else if c1, ok := cmdable.(clientContextable); ok {
		return c1.WithContext(ctx)
	} else if c1, ok := cmdable.(ringContextable); ok {
		return c1.WithContext(ctx)
	} else if c1, ok := cmdable.(clusterContextable); ok {
		return c1.WithContext(ctx)
	} else {
		panic(sderr.New("convert to Contextable error"))
	}
}
