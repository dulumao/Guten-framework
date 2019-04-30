package template

import (
	"bytes"
	"fmt"
	"github.com/CloudyKit/jet"
	"github.com/dulumao/Guten-framework/app/core/adapter/session"
	"github.com/dulumao/Guten-framework/app/core/adapter/validation"
	"github.com/dulumao/Guten-framework/app/core/env"
	"github.com/dulumao/Guten-framework/app/core/helpers/view"
	"github.com/dulumao/Guten-framework/app/core/observer"
	"github.com/dulumao/Guten-utils/conv"
	"github.com/dulumao/Guten-utils/dump"
	"github.com/dulumao/Guten-utils/file"
	"github.com/dulumao/Guten-utils/safemap"
	"github.com/labstack/echo"
	"html/template"
	"io"
	"strings"
	"time"
)

// 模板注册
type Renderer struct {
	Cached bool
	Engine *jet.Set
}

func New(cached bool, dirs ...string) *Renderer {
	return &Renderer{
		Cached: false,
		// Engine: jet.NewHTMLSet(dirs...),
		Engine: NewSetLoader(template.HTMLEscape, dirs...),
	}
}

func NewSetLoader(escapee jet.SafeWriter, dirs ...string) *jet.Set {
	return jet.NewSetLoader(escapee, &OSFileSystemLoader{dirs: dirs})
}

func (self *Renderer) Render(out io.Writer, name string, data interface{}, ctx echo.Context) error {
	self.Engine.SetDevelopmentMode(!self.Cached)

	self.Engine.AddGlobal("isLast", func(i, size int) bool { return i == size-1 })
	self.Engine.AddGlobal("isNotLast", func(i, size int) bool { return i != size-1 })
	self.Engine.AddGlobal("route", ctx.Echo().Reverse)
	self.Engine.AddGlobal("printf", fmt.Sprintf)
	self.Engine.AddGlobal("isNil", func(v interface{}) bool {
		return v == nil
	})
	self.Engine.AddGlobal("formatTime", func(layout string, v interface{}) string {
		switch t := v.(type) {
		case time.Time:
			return t.Format(layout)
		case *time.Time:
			return t.Format(layout)
		default:
			return ""
		}
	})
	self.Engine.AddGlobal("getCsrf", func(context echo.Context) string {
		return conv.String(context.Get("csrf"))
	})
	self.Engine.AddGlobal("isEqual", func(v1 interface{}, v2 interface{}) bool {
		if v1 == v2 {
			return true
		}

		return false
	})
	self.Engine.AddGlobal("session", func(key string) interface{} {
		sess := session.Default(ctx)
		data := sess.Get(key)
		sess.Save()

		return data
	})
	self.Engine.AddGlobal("flash", func(key string) []interface{} {
		sess := session.Default(ctx)
		data := sess.Flashes(key)
		sess.Save()

		return data
	})
	// jet has `unsafe` func
	// self.Engine.AddGlobal("unescaped", func(x string) interface{} {
	// 	return template.HTML(x)
	// })
	self.Engine.AddGlobal("GetFileSize", func(v uint64) interface{} {
		return file.FileSize(int64(v))
	})
	self.Engine.AddGlobal("dump", func(i ...interface{}) {
		dump.DD(i...)
	})
	self.Engine.AddGlobal("dump2", func(i ...interface{}) {
		dump.DD2(i...)
	})
	self.Engine.AddGlobal("HasValidError", func(key string, context echo.Context) bool {
		var errs = context.Get("errors")

		if validErrors, can := errs.(error); can {
			validResult := validation.GetErrorFields(validErrors, nil)

			if _, ok := validResult[key]; ok {
				return true
			}
		}

		return false
	})
	self.Engine.AddGlobal("GetValidError", func(key string, context echo.Context) string {
		var errs = context.Get("errors")
		// uni := v.GetTranslator("zh_Hans_CN")
		// uni := v.GetTranslator("en_US")
		// validResult := validation.GetErrorFields(err, uni)

		if validErrors, can := errs.(error); can {
			validResult := validation.GetErrorFields(validErrors, nil)

			if _, ok := validResult[key]; ok {
				return validResult[key]["transText"]
			}
		}

		return ""
	})

	for k, v := range view.Funcs.Items() {
		self.Engine.AddGlobal(conv.String(k), v)
	}

	t, err := self.Engine.GetTemplate(name)

	if err != nil {
		panic(err)
	}

	vars := make(jet.VarMap)
	vars.Set("context", ctx)
	vars.Set("env", env.Value)

	buf := new(bytes.Buffer)

	// if err = t.Execute(out, vars, data); err != nil {
	// 	panic(err)
	// }
	//
	if err = t.Execute(buf, vars, data); err != nil {
		panic(err)
	}

	eventArgs := safemap.NewSafeMap()
	eventArgs.Set("name", name)
	eventArgs.Set("data", data)
	eventArgs.Set("context", ctx)
	eventArgs.Set("buf", buf.Bytes())

	observer.Dispatcher.Emit("view.after."+GetViewEventName(name), eventArgs)

	if buf, ok := eventArgs.Get("buf").([]byte); ok {
		out.Write(buf)
	} else if buf, ok := eventArgs.Get("buf").(string); ok {
		out.Write([]byte(buf))
	}

	return err
}

func GetViewEventName(name string) string {
	var names = strings.Split(name, "/")

	return strings.Join(names, ".")
}
