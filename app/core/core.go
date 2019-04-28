package core

import (
	"fmt"
	"github.com/dulumao/Guten-framework/app/core/adapter/cache"
	"github.com/dulumao/Guten-framework/app/core/adapter/context"
	"github.com/dulumao/Guten-framework/app/core/adapter/database"
	"github.com/dulumao/Guten-framework/app/core/adapter/session"
	"github.com/dulumao/Guten-framework/app/core/adapter/template"
	"github.com/dulumao/Guten-framework/app/core/adapter/validation"
	"github.com/dulumao/Guten-framework/app/core/env"
	CoreMiddleware "github.com/dulumao/Guten-framework/app/core/middleware"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/fatih/color"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/labstack/gommon/random"
	"net/http"
	"os"
)

func New() *echo.Echo {
	env.New()
	cache.New()

	app := echo.New()

	app.HideBanner = true

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
			cc := &context.Context{Context: c}

			cc.SetCodeCompiledTimeAt()

			return next(cc)
		}
	})

	app.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
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

		if !app.Debug {
			app.Logger.Print(err.Error())

			if context.Request().Header.Get("X-Requested-With") == "xmlhttprequest" {
				context.JSON(http.StatusInternalServerError, map[string]interface{}{
					"message": err.Error(),
				})
			} else if context.Request().Method == echo.HEAD { // Issue #608 {
				err = context.NoContent(http.StatusInternalServerError)
			} else {
				err = context.Render(http.StatusOK, "error/fail", map[string]interface{}{
					"message": message,
				})
			}
		}
		/*if app.Debug {
			if !context.Response().Committed {
				if context.Request().Header.Get("X-Requested-With") == "xmlhttprequest" {
					context.JSON(code, map[string]interface{}{
						"message": err.Error(),
					})
				} else if context.Request().Method == echo.HEAD { // Issue #608 {
					err = context.NoContent(code)
				} else {
					err = context.HTML(code, conv.String(message))
				}
			}
		} else {
			err = context.Render(http.StatusOK, "error/fail", nil)
		}*/

		// app.Logger.Error(err)
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
