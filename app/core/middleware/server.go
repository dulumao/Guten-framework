package middleware

import (
	"github.com/dulumao/Guten-framework/app/core/constant"
	"fmt"
	"github.com/labstack/echo"
)

func Server(next echo.HandlerFunc) echo.HandlerFunc {
	return func(context echo.Context) error {
		context.Response().Header().Set(echo.HeaderServer, fmt.Sprintf("%s/%s", constant.ServerName, constant.ServerVersion))
		return next(context)
	}
}
