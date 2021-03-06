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
	"runtime"
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

					// frames := stack.Callers(0)
					// frames := stack.Callers(4)

					// depthOfPanic := findPanic()
					depthOfTemplateError := findTemplateError()
					skipFrames := 0

					/*if depthOfPanic != 0 {
						skipFrames = depthOfPanic + 1
					} else {
						skipFrames = findTemplateError() + 1
					}*/

					if depthOfTemplateError != 0 {
						skipFrames = depthOfTemplateError - 1
					} else {
						skipFrames = findPanic() + 1
					}

					frames := stack.Callers(skipFrames)

					if env.Value.Server.Debug {
						_ = fmt.Sprintf(logFmt, levelColor("PANIC")+pathColor(" at "+c.Request().URL.Path), err, frames.String())
					}

					file := appendGOPATH(frames[0].File)
					src := mustReadLines(file)

					start := (frames[0].Line - 1) - 5
					// end := frames[0].Line + 5
					end := frames[0].Line + 2
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

func findPanic() int {
	stack := make([]uintptr, 50)
	// skip two frames: runtime.Callers + findPanic
	nCallers := runtime.Callers(2, stack[:])
	frames := runtime.CallersFrames(stack[:nCallers])
	var skipNumber = 0

	for i := 0; ; i++ {
		frame, more := frames.Next()

		if frame.Function == "github.com/gookit/validate.(*Validation).Validate" {
			skipNumber = i
		}

		if frame.Function == "runtime.gopanic" {
			skipNumber = i
		}

		if frame.Function == "runtime.sigpanic" {
			skipNumber = i
		}

		if !more {
			return skipNumber
		}
	}

	return skipNumber
}

func findTemplateError() int {
	stack := make([]uintptr, 12)
	nCallers := runtime.Callers(0, stack)
	frames := runtime.CallersFrames(stack[:nCallers])

	for i := 0; ; i++ {
		frame, more := frames.Next()
		_ = more
		// Function: (string) (len=42) "github.com/labstack/echo.(*context).Render",
		if strings.Contains(frame.Function, "github.com/labstack/echo.(*context).Render") {
			return i
		} else if !more {
			// Exhausted the stack, take deepest.
			return 0
		}
	}
}
