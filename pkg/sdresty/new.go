package sdresty

import (
	"crypto/tls"
	"time"

	"github.com/go-resty/resty/v2"
)

type Options struct {
	Timeout            time.Duration
	RetryCount         int
	Proxy              string
	InsecureSkipVerify bool
	QueryParams        map[string]string
	PathParams         map[string]string
	Headers            map[string]string
}

func New(opts Options) *resty.Client {
	c := resty.New()
	c.SetTimeout(opts.Timeout)
	c.SetRetryCount(opts.RetryCount)
	if opts.Proxy != "" {
		c.SetProxy(opts.Proxy)
	} else {
		c.RemoveProxy()
	}
	if opts.InsecureSkipVerify {
		c.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}
	if len(opts.QueryParams) > 0 {
		c.SetQueryParams(opts.QueryParams)
	}
	if len(opts.PathParams) > 0 {
		c.SetPathParams(opts.PathParams)
	}
	if len(opts.Headers) > 0 {
		c.SetHeaders(opts.Headers)
	}
	c.SetDebug(true)
	return c
}
