package database

import (
	"fmt"
	"github.com/dulumao/Guten-framework/app/core/env"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/labstack/echo"
	"net/url"
	// _ "github.com/jinzhu/gorm/dialects/sqlite"
	// _ "github.com/jinzhu/gorm/dialects/mssql"
	"github.com/jinzhu/gorm"
)

var DB *gorm.DB

func New(app *echo.Echo) {
	var err error

	if env.Value.Database.Driver == "mysql" {
		DB, err = gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=%s",
			env.Value.Database.Mysql.Username,
			env.Value.Database.Mysql.Password,
			env.Value.Database.Mysql.Host,
			env.Value.Database.Mysql.Port,
			env.Value.Database.Mysql.Database,
			env.Value.Database.Mysql.Charset,
			url.QueryEscape(env.Value.Server.Timezone),
		))

		if err != nil {
			panic(err)
		}

		DB.SingularTable(true)
		DB.LogMode(env.Value.Database.Debug)

		// 不使用系统 log,否则将无法格式化sql参数，gorm有自己的格式化
		// db.SetLogger(log.New(vars.Kernel.LogWriter, "\r\n", 0))
		// DB.SetLogger(app.Logger)
		// DB.SetLogger(gorm.Logger{app.Logger})

		// DB.Callback().Create().Replace("gorm:update_time_stamp", updateTimeStampForCreateCallback)
		// DB.Callback().Update().Replace("gorm:update_time_stamp", updateTimeStampForUpdateCallback)
		// DB.Callback().Delete().Replace("gorm:delete", deleteCallback)

		DB.DB().SetMaxOpenConns(env.Value.Database.MaxOpen)
		DB.DB().SetMaxIdleConns(env.Value.Database.MaxIdle)
	}

	if env.Value.Database.Debug {
		DB = DB.Debug()
	}
}

func CloseDB() {
	defer DB.Close()
}
