package context

import (
	"github.com/dulumao/Guten-framework/app/core/adapter/auth"
	"github.com/dulumao/Guten-framework/app/core/adapter/session"
	"github.com/dulumao/Guten-utils/conv"
	"github.com/labstack/echo"
	"strings"
	"time"
)

type Context struct {
	echo.Context
	codeCompiledTimeAt time.Time
}

func (self *Context) GetSession() session.ISession {
	return session.Default(self.Context)
}

func (self *Context) GetAuth() *auth.AuthManager {
	return auth.New().SetContext(self.Context)
}

func (self *Context) SetCodeCompiledTimeAt() {
	self.codeCompiledTimeAt = time.Now()
}

func (self *Context) GetElapsed() time.Duration {
	elapsed := time.Since(self.codeCompiledTimeAt)

	return elapsed
}

func (self *Context) ParamInt(name string) int {
	return conv.Int(self.Param(name))
}

func (self *Context) ParamInt8(name string) int8 {
	return conv.Int8(self.Param(name))
}

func (self *Context) ParamInt16(name string) int16 {
	return conv.Int16(self.Param(name))
}

func (self *Context) ParamInt64(name string) int64 {
	return conv.Int64(self.Param(name))
}

func (self *Context) ParamUint(name string) uint {
	return conv.Uint(self.Param(name))
}

func (self *Context) ParamUint8(name string) uint8 {
	return conv.Uint8(self.Param(name))
}

func (self *Context) ParamUint16(name string) uint16 {
	return conv.Uint16(self.Param(name))
}

func (self *Context) ParamUint64(name string) uint64 {
	return conv.Uint64(self.Param(name))
}

func (self *Context) HasParam(name string) bool {
	v := self.Param(name)

	if v == "" {
		return false
	}

	return true
}

func (self *Context) GetParam(name string) (string, bool) {
	v := self.Param(name)

	if v == "" {
		return "", false
	}

	return v, true
}

func (self *Context) IsPost() bool {
	return self.Request().Method == echo.POST
}

func (self *Context) IsGet() bool {
	return self.Request().Method == echo.GET
}

func (self *Context) IsDelete() bool {
	return self.Request().Method == echo.DELETE
}

func (self *Context) IsPath() bool {
	return self.Request().Method == echo.PATCH
}

func (self *Context) IsPut() bool {
	return self.Request().Method == echo.PUT
}

func (self *Context) IsAjax() bool {
	return strings.ToLower(self.Request().Header.Get("X-Requested-With")) == "xmlhttprequest"
}
