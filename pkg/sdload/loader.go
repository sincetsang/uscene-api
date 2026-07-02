package sdload

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"TMA/pkg/sderr"
	"TMA/pkg/sdresty"
)

// Loader
type Loader interface {
	LoadBytes(loc string) ([]byte, error)
}

// LoaderFunc
type LoaderFunc func(loc string) ([]byte, error)

func (f LoaderFunc) LoadBytes(loc string) ([]byte, error) {
	return f(loc)
}

// Loaders
var (
	loaders = map[string]Loader{
		"":      LoaderFunc(fileLoader),
		"file":  LoaderFunc(fileLoader),
		"f":     LoaderFunc(fileLoader),
		"http":  LoaderFunc(httpLoader),
		"https": LoaderFunc(httpLoader),
	}
)

func RegisterLoader(scheme string, l Loader) {
	if l == nil {
		return
	}
	loaders[scheme] = l
}

// default loader

func fileLoader(loc string) ([]byte, error) {
	loc = strings.TrimPrefix(loc, "file://")
	data, err := ioutil.ReadFile(loc)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	return data, nil
}

var (
	httpResty = sdresty.New(sdresty.Options{
		Timeout:            10 * time.Second,
		InsecureSkipVerify: true,
		RetryCount:         1,
	})
)

func httpLoader(loc string) ([]byte, error) {
	resp, err := httpResty.R().Get(loc)
	if err != nil {
		return nil, sderr.WithStack(err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, sderr.Newf("response HTTP status error: %d, '%s'", resp.StatusCode(), loc)
	}
	data := resp.Body()
	return data, nil
}
