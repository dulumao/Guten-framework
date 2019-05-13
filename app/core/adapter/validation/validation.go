package validation

import (
	"github.com/gookit/validate"
	"github.com/labstack/echo"
)

type Validation struct {
	scene []string
}

var Validator *Validation

func New(scene ...string) *Validation {
	validate.Config(func(opt *validate.GlobalOption) {
		opt.FilterTag = "filter"
		opt.ValidateTag = "valid"
		opt.StopOnError = false
		opt.SkipOnEmpty = true
	})

	Validator = &Validation{
		scene: scene,
	}

	return Validator
}

func (self *Validation) Config(fn func(opt *validate.GlobalOption)) {
	validate.Config(fn)
}

// validate.Errors
func (self *Validation) Validate(i interface{}) error {
	v := validate.Struct(i)

	if v.Validate() {
		return nil
	}

	return v.Errors
}

func (self *Validation) Map(v map[string]interface{}) *validate.Validation {
	return validate.Map(v, self.scene...)
}

func (self *Validation) JSON(v string) *validate.Validation {
	return validate.JSON(v, self.scene...)
}

func (self *Validation) Request(c echo.Context) *validate.Validation {
	r := c.Request()

	return validate.Request(r)
}
