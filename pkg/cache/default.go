package memory

import (
	"TMA/pkg/nucl"
	"TMA/pkg/sderr"
	"context"
	"sync"
	"time"
)

type DataGenerator func(ctx context.Context) (interface{}, error)
type DataSetter func(data interface{})

type DefaultCache struct {
	sync.RWMutex
	UpdatePeriod  time.Duration
	Nu            *nucl.Nucleus
	CacheName     string
	DataGenerator DataGenerator
	DataSetter    DataSetter
}

func (c *DefaultCache) Init(ctx context.Context) error {
	data, err := c.DataGenerator(ctx)
	if err != nil {
		return sderr.WithStack(err)
	}
	c.Lock()
	defer c.Unlock()
	c.DataSetter(data)
	return nil
}

func (c *DefaultCache) Update(ctx context.Context) error {
	data, err := c.DataGenerator(ctx)
	if err != nil {
		return sderr.WithStack(err)
	}
	c.Lock()
	defer c.Unlock()
	c.DataSetter(data)
	return nil
}

func (c *DefaultCache) Name() string {
	return c.CacheName
}

func (c *DefaultCache) Timer() time.Duration {
	return c.UpdatePeriod
}
