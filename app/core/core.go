package core

import (
	"context"
	"fmt"
	"github.com/dulumao/Guten-framework/app/core/adapter/cache"
	CoreContext "github.com/dulumao/Guten-framework/app/core/adapter/context"
	"github.com/dulumao/Guten-framework/app/core/adapter/database"
	"github.com/dulumao/Guten-framework/app/core/adapter/session"
	"github.com/dulumao/Guten-framework/app/core/adapter/template"
	"github.com/dulumao/Guten-framework/app/core/adapter/validation"
	"github.com/dulumao/Guten-framework/app/core/env"
	CoreMiddleware "github.com/dulumao/Guten-framework/app/core/middleware"
	"github.com/dulumao/Guten-framework/app/core/observer"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/fatih/color"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/labstack/gommon/random"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

func New() *echo.Echo {
	env.New()
	cache.New()
	observer.New()

	app := echo.New()

	app.HideBanner = true
	app.Debug = env.Value.Server.Debug

	if l, ok := app.Logger.(*log.Logger); ok {
		// l.SetHeader(`(${long_file}:${line}) ${level}`)
		l.SetHeader("[L${line}: ${long_file}] ${time_rfc3339} ${message}")
	}

	app.Logger.SetLevel(log.OFF)

	if env.Value.Server.LogLevel == "DEBUG" {
		app.Logger.SetLevel(log.DEBUG)
	}

	if env.Value.Server.LogLevel == "INFO" {
		app.Logger.SetLevel(log.INFO)
	}

	if env.Value.Server.LogLevel == "ERROR" {
		app.Logger.SetLevel(log.ERROR)
	}

	if env.Value.Server.LogLevel == "WARN" {
		app.Logger.SetLevel(log.ERROR)
	}

	if !env.Value.Server.Debug {
		if fd, err := os.OpenFile(env.Value.Server.LogFile, os.O_RDWR|os.O_CREATE, 0755); err != nil {
			app.Logger.Panic(err)
		} else {
			app.Logger.SetOutput(fd)
		}
	}

	app.Pre(middleware.MethodOverride())

	if env.Value.Server.Debug {
		app.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			// Format: "[${method} ${status}]::${uri} [ERROR]::${error}\n",
			Format: "[${status}] ${method} ${uri}\n",
		}))
	}

	app.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string {
			return random.String(32)
		},
	}))

	app.Use(CoreMiddleware.Recover())
	app.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))
	app.Use(middleware.BodyLimit("20M"))

	var store session.IStore

	if env.Value.Session.Driver == "cookie" {
		store = session.NewCookieStore([]byte(env.Value.Server.HashKey))
	}

	if env.Value.Session.Driver == "file" {
		store = session.NewFilesystemStore(env.Value.Session.File.Path, []byte(env.Value.Server.HashKey))
	}

	if env.Value.Session.Driver == "redis" {
		var err error

		store, err = session.NewRedisStore(32, "tcp", env.Value.Session.Redis.Addr, env.Value.Session.Redis.Password, []byte(env.Value.Server.HashKey))

		if err != nil {
			app.Logger.Panic(err)
		}
	}

	store.Options(session.Options{
		Path:     env.Value.Session.Path,
		MaxAge:   env.Value.Session.Lifetime,
		HttpOnly: env.Value.Session.HTTPOnly,
		Secure:   env.Value.Session.Secure,
	})

	app.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &CoreContext.Context{Context: c}

			cc.SetCodeCompiledTimeAt()

			return next(cc)
		}
	})

	app.Use(CoreMiddleware.CSRFWithConfig(CoreMiddleware.CSRFConfig{
		TokenLookup: "form:_token",
		Skipper: func(context echo.Context) bool {
			// paths := strings.Split(strings.TrimPrefix(context.Request().URL.Path, "/"), "/")
			// if paths[0] == "" {
			// 	return true
			// }

			return false
		},
	}))

	app.Use(session.New(env.Value.Session.Name, store))

	app.Renderer = template.New(false, env.Value.Framework.TemplateDirs...)
	app.Validator = validation.New()

	app.HTTPErrorHandler = func(err error, context echo.Context) {
		// var code = http.StatusInternalServerError
		var message interface{}

		if http, ok := err.(*echo.HTTPError); ok {
			// code = http.Code
			message = http.Message

			if http.Internal != nil {
				err = fmt.Errorf("%v, %v", err, http.Internal)
			}
		} else {
			message = err.Error()
		}

		if !env.Value.Server.Debug {
			message = "发生致命错误了"
			app.Logger.Error(err.Error())
		}

		if !context.Response().Committed {
			if context.Request().Header.Get("X-Requested-With") == "xmlhttprequest" {
				context.JSON(http.StatusInternalServerError, map[string]interface{}{
					"message": message,
				})
			} else if context.Request().Method == echo.HEAD { // Issue #608 {
				err = context.NoContent(http.StatusInternalServerError)
			} else {
				err = context.Render(http.StatusOK, "error/fail", map[string]interface{}{
					"message": message,
				})
			}
		}
	}

	app.Static("/template", "web/assets/template")
	app.Static("/uploads", "web/assets/uploads")
	app.Static("/static", "web/assets/static")

	echo.NotFoundHandler = func(c echo.Context) error {
		return c.Render(http.StatusNotFound, "error/not_found", nil)
	}

	database.New(app)

	app.Server.Addr = env.Value.Server.Addr

	color.New(color.FgWhite).Println("\nServer running at:")
	color.New(color.FgWhite).Print("- ")
	color.New(color.FgGreen).Println(fmt.Sprintf("http://%s", app.Server.Addr))

	return app
}

func Serve(app *echo.Echo) {
	app.Logger.Panic(gracehttp.Serve(app.Server))
}

func GracefulServe(app *echo.Echo) {
	go func() {
		if err := app.Start(app.Server.Addr); err != nil {
			app.Logger.Errorf("Shutting down the server with error:%v", err)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	if err := app.Server.Shutdown(ctx); err != nil {
		app.Logger.Fatal(err)
	}
}

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))

	if err != nil {
		panic(err)
	}

	return strings.Replace(dir, "\\", "/", -1)
}
