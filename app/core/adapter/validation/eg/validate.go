package main

import (
	"fmt"
	"github.com/dulumao/Guten-utils/dump"
	"github.com/gookit/validate"
	"time"
)

// UserForm struct
type UserForm struct {
	Name              string    `filter:"upper" valid:"required|minLen:7"`
	Email             string    `valid:"email"`
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
	}
}

// Translates you can custom field translates.
func (f UserForm) Translates() map[string]string {
	return validate.MS{
		"Name":     "User Name",
		"Email":    "User Email",
		"Password": "密码",
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
	}

	validate.Config(func(opt *validate.GlobalOption) {
		opt.FilterTag = "filter"
		opt.ValidateTag = "valid"
		opt.StopOnError = true
		opt.SkipOnEmpty = true
	})

	v := validate.Struct(u)
	dump.DD(u)
	if v.Validate() { // validate ok
		// do something ...
	} else {
		fmt.Println(v.Errors.All()) // all error messages
		// fmt.Println(v.Errors.One()) // returns a random error message text
		// fmt.Println(v.Errors.Field("Name")) // returns error messages of the field
	}
}
