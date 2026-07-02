package sdcall

import (
	"github.com/panjf2000/ants/v2"
	"sync"

	"TMA/pkg/sderr"
)

type Pool struct {
	pool *ants.Pool
}

type PoolOptions = ants.Options

var (
	ErrInvalidPoolExpiry   = ants.ErrInvalidPoolExpiry
	ErrLackPoolFunc        = ants.ErrLackPoolFunc
	ErrPoolClosed          = ants.ErrPoolClosed
	ErrPoolOverload        = ants.ErrPoolOverload
	ErrInvalidPreAllocSize = ants.ErrInvalidPreAllocSize
)

func NewPool(size int, opts *PoolOptions) (*Pool, error) {
	antsOpts := []ants.Option{}
	if opts != nil {
		antsOpts = append(antsOpts, ants.WithOptions(*opts))
	}
	p, err := ants.NewPool(size, antsOpts...)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	return &Pool{pool: p}, nil
}

func (p *Pool) NumFree() int {
	return p.pool.Free()
}

func (p *Pool) NumCap() int {
	return p.pool.Cap()
}

func (p *Pool) NumRunning() int {
	return p.pool.Running()
}

func (p *Pool) Close() error {
	p.pool.Release()
	return nil
}

func (p *Pool) Submit(f func()) error {
	if f == nil {
		return nil
	}
	err := p.pool.Submit(f)
	return sderr.WithStack(err)
}

func (p *Pool) Do(f func()) error {
	if f == nil {
		return nil
	}
	var wg sync.WaitGroup
	wg.Add(1)
	err := p.pool.Submit(func() {
		defer wg.Done()
		_ = Safe(f)
	})
	if err != nil {
		return sderr.WithStack(err)
	}
	wg.Wait()
	return nil
}

func (p *Pool) Wrap(f func()) func() {
	if f == nil {
		return nil
	}
	return func() {
		_ = p.Do(f)
	}
}
