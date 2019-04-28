package context

import (
	"github.com/dulumao/Guten-framework/app/core/adapter/auth"
	"github.com/dulumao/Guten-framework/app/core/adapter/session"
	"github.com/dulumao/Guten-utils/conv"
	"github.com/labstack/echo"
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
	return self.Request().Header.Get("X-Requested-With") == "xmlhttprequest"
}
