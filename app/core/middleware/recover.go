package middleware

import (
	"fmt"
	"github.com/dulumao/Guten-framework/app/core/env"
	"github.com/facebookgo/stack"
	"github.com/fatih/color"
	"github.com/labstack/echo"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var gopaths []string

func init() {
	for _, p := range strings.Split(os.Getenv("GOPATH"), ":") {
		if p != "" {
			gopaths = append(gopaths, filepath.Join(p, "src")+"/")
		}
	}
}

func mustReadLines(filename string) []string {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return strings.Split(string(bytes), "\n")
}

func appendGOPATH(file string) string {
	for _, p := range gopaths {
		f := filepath.Join(p, file)
		if _, err := os.Stat(f); err == nil {
			return f
		}
	}
	return file
}

func Recover() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			defer func() {
				if r := recover(); r != nil {
					var logFmt = "\n[%s]\n\n%v\n\nStack Trace:\n%s\n"
					var levelColor = color.New(color.Bold, color.FgRed).SprintFunc()
					var pathColor = color.New(color.Reset, color.FgRed).SprintFunc()

					err, ok := r.(error)

					if !ok {
						err = fmt.Errorf("%v", r)
					}

					frames := stack.Callers(4)

					if env.Value.Server.Debug {
						_ = fmt.Sprintf(logFmt, levelColor("PANIC")+pathColor(" at "+c.Request().URL.Path), err, frames.String())
					}

					file := appendGOPATH(frames[0].File)
					src := mustReadLines(file)

					start := (frames[0].Line - 1) - 5
					end := frames[0].Line + 5
					lines := src[start:end]

					// c.Error(err)
					if env.Value.Server.Debug {
						if c.Request().Header.Get("X-Requested-With") == "xmlhttprequest" {
							c.JSON(http.StatusInternalServerError, map[string]interface{}{
								"message":  err.Error(),
								"location": frames[0].String(),
							})
						} else {
							c.Render(http.StatusOK, "error/panic", map[string]interface{}{
								"URL":         c.Request().URL.Path,
								"Err":         err,
								"Name":        frames[0].Name,
								"File":        frames[0].File,
								"StartLine":   start + 1,
								"SourceLines": strings.Join(lines, "\n"),
								"Frames":      frames,
							})
						}
					} else {
						c.Error(err)
					}
				}
			}()

			return next(c)
		}
	}
}
