package web

import (
	"TMA/pkg/middleware"
	"TMA/pkg/sderr"
	"TMA/pkg/sdjson"
	"TMA/pkg/sdtime"
	"TMA/pkg/web/webapi"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

const (
	defaultIndexPage = "index.html"
)

type EchoOptions struct {
	LogSkipper   echoMiddleware.Skipper
	ErrorHandler echo.HTTPErrorHandler
}

func pingAPI(ec echo.Context) error {
	data := sdjson.Object{
		"pong": sdtime.NowUnixMs(),
	}
	return webapi.OK(data).Render(ec)
}

func NewEcho(opts EchoOptions) *echo.Echo {
	if opts.ErrorHandler == nil {
		opts.ErrorHandler = defaultHttpErrorHandler
	}
	app := echo.New()
	app.HideBanner = true
	app.HidePort = true
	app.Use(middleware.LoggingRecover(opts.LogSkipper))
	app.HTTPErrorHandler = opts.ErrorHandler
	// 用于安全监测
	app.HEAD("/", pingAPI)
	return app
}

func NoRedirectStatic(app *echo.Echo, pathPrefix, fsRoot string) *echo.Route {
	subFs := echo.MustSubFS(app.Filesystem, fsRoot)
	return app.Add(
		http.MethodGet,
		pathPrefix+"*",
		NoRedirectStaticDirectoryHandler(subFs, false),
	)
}

func NoRedirectStaticDirectoryHandler(fileSystem fs.FS, disablePathUnescaping bool) echo.HandlerFunc {
	return func(ec echo.Context) error {
		p := ec.Param("*")
		if !disablePathUnescaping {
			tmpPath, err := url.PathUnescape(p)
			if err != nil {
				return fmt.Errorf("failed to unescape path variable: %w", err)
			}
			p = tmpPath
		}

		name := filepath.ToSlash(filepath.Clean(strings.TrimPrefix(p, "/")))
		fi, err := fs.Stat(fileSystem, name)
		if err != nil {
			return echo.ErrNotFound
		}

		p = ec.Request().URL.Path
		if fi.IsDir() {
			name = defaultIndexPage
		}
		return fsFile2(ec, name, fileSystem)
	}
}

func fsFile2(ec echo.Context, file string, filesystem fs.FS) error {
	f, err := filesystem.Open(file)
	if err != nil {
		return echo.ErrNotFound
	}
	defer func() { _ = f.Close() }()

	fi, _ := f.Stat()
	if fi.IsDir() {
		file = filepath.ToSlash(filepath.Join(file, defaultIndexPage))
		f, err = filesystem.Open(file)
		if err != nil {
			return echo.ErrNotFound
		}
		defer func() { _ = f.Close() }()
		if fi, err = f.Stat(); err != nil {
			return err
		}
	}
	ff, ok := f.(io.ReadSeeker)
	if !ok {
		return sderr.New("file does not implement io.ReadSeeker")
	}
	http.ServeContent(ec.Response(), ec.Request(), fi.Name(), fi.ModTime(), ff)
	return nil
}
