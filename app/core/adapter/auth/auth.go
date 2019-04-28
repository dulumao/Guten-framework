package auth

import (
	"github.com/dulumao/Guten-framework/app/core/adapter/session"
	"github.com/dulumao/Guten-utils/conv"
	"github.com/labstack/echo"
)

type IAuth interface {
	GetId() int
	GetUserName() string
	GetGuard() string
	GetUser(int)
}

type AuthManager struct {
	Session session.ISession
}

func New() *AuthManager {
	return new(AuthManager)
}

func (self *AuthManager) SetAttempt(auth IAuth) bool {
	self.Session.Set("auth_"+auth.GetGuard(), auth.GetId())

	if err := self.Session.Save(); err != nil {
		panic(err)
		return false
	}

	return true
}

func (self AuthManager) SetContext(context echo.Context) *AuthManager {
	self.Session = session.Default(context)

	return &self
}

func (self *AuthManager) Guest(auth IAuth) bool {
	uid := self.Session.Get("auth_" + auth.GetGuard())

	if uid == nil {
		return true
	}

	return false
}

func (self *AuthManager) User(auth IAuth) {
	uid := self.Session.Get("auth_" + auth.GetGuard())

	auth.GetUser(conv.Int(uid))
}

func (self AuthManager) Logout(auth IAuth) {
	self.Session.Delete("auth_" + auth.GetGuard())
	self.Session.Save()
}
