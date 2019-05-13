package main

import (
	"github.com/dulumao/Guten-framework/app/core/adapter/validation"
	"github.com/dulumao/Guten-utils/dump"
	"github.com/gookit/validate"
	"time"
)

// UserForm struct
type UserForm struct {
	Name              string    `filter:"upper" valid:"required|minLen:7"`
	Avatar            string    `filter:"upper" valid:"required"`
	Email             string    `valid:"email"`
	QQ                string    `valid:"required|qq"`
	Age               int       `valid:"required|int|min:1|max:99"`
	CreateAt          int       `valid:"min:1"`
	Safe              int       `valid:"-"`
	UpdateAt          time.Time `valid:"required"`
	Code              string    `valid:"required|customValidator"`
	Password          string    `valid:"required|uint"`
	ConfirmedPassword string    `valid:"required"`
	Test              string    `filter:"upper" valid:"required|in:ONE,TWO,THREE" `
}

// CustomValidator custom validator in the source struct.
func (f UserForm) CustomValidator(val string) bool {
	return len(val) == 4
}

// Messages you can custom validator error messages.
func (f UserForm) Messages() map[string]string {
	return validate.MS{
		"required":                  "oh! the {field} is required",
		"Name.required":             "message for special field",
		"Password.isUint":           "{field} 必须是整数",
		"ConfirmedPassword.eqField": "确认密码必须相等",
		"QQ.qq":                     "{field}不符合规则",
	}
}

// Translates you can custom field translates.
func (f UserForm) Translates() map[string]string {
	return validate.MS{
		"Name":     "用户名",
		"Email":    "邮箱",
		"Password": "密码",
		"QQ":       "QQ号",
	}
}

func main() {
	u := &UserForm{
		Name:              "mattma1",
		Age:               99,
		Code:              "9999",
		UpdateAt:          time.Now(),
		Password:          "111a",
		ConfirmedPassword: "111a",
		Test:              "one",
		QQ:                "test",
	}

	validation.New()

	err := validation.Validator.Validate(u)

	if err != nil {
		// panic(err)
		errs := err.(validate.Errors)
		dump.DD(errs.All())

		// t := validate.NewTranslator()
		// t.LoadMessages(locales.Locales["zh-CN"])
		// t.AddFieldMap(validate.MS{
		// 	"Avatar.required": "1111",
		// })
		//
		// dump.DD(t.FieldMap())

		return
	}

	dump.DD("pass")

	// v := validate.Struct(u)
	// if v.Validate() { // validate ok
	// 	// do something ...
	// } else {
	// 	fmt.Println(v.Errors.All()) // all error messages
	// 	// fmt.Println(v.Errors.One()) // returns a random error message text
	// 	// fmt.Println(v.Errors.Field("Name")) // returns error messages of the field
	// }
}
