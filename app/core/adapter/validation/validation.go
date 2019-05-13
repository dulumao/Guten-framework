package validation

import (
	"github.com/gookit/validate"
	"github.com/gookit/validate/locales"
	"github.com/labstack/echo"
	"regexp"
)

type Validation struct {
	scene []string
}

var Validator *Validation

func New(scene ...string) {
	// vv := validate.NewEmpty()

	// vv.ValidateData()
	validate.Config(func(opt *validate.GlobalOption) {
		opt.FilterTag = "filter"
		opt.ValidateTag = "valid"
		opt.StopOnError = false
		opt.SkipOnEmpty = true
	})

	v := &Validation{
		scene: scene,
	}

	v.addValidations()

	Validator = v
}

func (self *Validation) Config(fn func(opt *validate.GlobalOption)) {
	validate.Config(fn)
}

// validate.Errors
func (self *Validation) Validate(i interface{}) error {
	v := validate.Struct(i)

	// locales.Register(v, "zh-CN")

	if v.Validate() {
		return nil
	}

	return v.Errors
}

func (self *Validation) Struct(i interface{}, lang interface{}) validate.Errors {
	v := validate.Struct(i)

	return self.getErrors(v, lang)
}

func (self *Validation) Map(i map[string]interface{}, lang interface{}) validate.Errors {
	v := validate.Map(i, self.scene...)

	return self.getErrors(v, lang)
}

func (self *Validation) JSON(i string, lang interface{}) validate.Errors {
	v := validate.JSON(i, self.scene...)

	return self.getErrors(v, lang)
}

func (self *Validation) Request(c echo.Context) validate.Errors {
	r := c.Request()
	v := validate.Request(r)

	return self.getErrors(v, nil)
}

func (self *Validation) Regexp(str string, pattern string) bool {
	return validate.Regexp(str, pattern)
}

func (self *Validation) getErrors(v *validate.Validation, lang interface{}) validate.Errors {
	if lang != nil {
		switch lang.(type) {
		case string:
			message := lang.(string)
			if locales, ok := locales.Locales[message]; ok {
				v.WithMessages(locales)
			}
		case map[string]string:
			locales := lang.(map[string]string)
			v.WithMessages(locales)
		}
	}

	if v.Validate() {
		return nil
	}

	return v.Errors
}

func (self *Validation) addValidations() {
	validate.AddValidator("phone", func(val string) bool {
		reg := regexp.MustCompile(`^13[\d]{9}$|^14[5,7]{1}\d{8}$|^15[^4]{1}\d{8}$|^17[0,3,5,6,7,8]{1}\d{8}$|^18[\d]{9}$`)

		return reg.MatchString(val)
	})
	validate.AddValidator("telephone", func(val string) bool {
		// 国内座机电话号码："XXXX-XXXXXXX"、"XXXX-XXXXXXXX"、"XXX-XXXXXXX"、"XXX-XXXXXXXX"、"XXXXXXX"、"XXXXXXXX"
		reg := regexp.MustCompile(`^((\d{3,4})|\d{3,4}-)?\d{7,8}$`)

		return reg.MatchString(val)
	})
	validate.AddValidator("qq", func(val string) bool {
		// 腾讯QQ号，从10000开始
		reg := regexp.MustCompile(`^[1-9][0-9]{4,}$`)

		return reg.MatchString(val)
	})
	validate.AddValidator("cnPostcode", func(val string) bool {
		// 中国邮政编码
		reg := regexp.MustCompile(`^\d{6}$`)

		return reg.MatchString(val)
	})
	validate.AddValidator("username", func(val string) bool {
		// 通用帐号规则(字母开头，只能包含字母、数字和下划线，长度在6~18之间)
		reg := regexp.MustCompile(`^[a-zA-Z]{1}\w{5,17}$`)

		return reg.MatchString(val)
	})
	validate.AddValidator("password", func(val string) bool {
		// 通用密码(任意可见字符，长度在6~18之间)
		reg := regexp.MustCompile(`^[\w\S]{6,18}$`)

		return reg.MatchString(val)
	})
	validate.AddValidator("password2", func(val string) bool {
		// 中等强度密码(在弱密码的基础上，必须包含大小写字母和数字)
		reg1 := regexp.MustCompile(`^[\w\S]{6,18}$`)
		reg2 := regexp.MustCompile(`[a-z]+`)
		reg3 := regexp.MustCompile(`[A-Z]+`)
		reg4 := regexp.MustCompile(`\d+`)

		return reg1.MatchString(val) && reg2.MatchString(val) && reg3.MatchString(val) && reg4.MatchString(val)
	})
	validate.AddValidator("cnIdNumber", func(val string) bool {
		/*
				公民身份证号
				xxxxxx yyyy MM dd 375 0     十八位
				xxxxxx   yy MM dd  75 0     十五位
				地区：[1-9]\d{5}
				年的前两位：(18|19|([23]\d))      1800-2399
				年的后两位：\d{2}
				月份：((0[1-9])|(10|11|12))
				天数：(([0-2][1-9])|10|20|30|31) 闰年不能禁止29+
				三位顺序码：\d{3}
				两位顺序码：\d{2}
				校验码：   [0-9Xx]
				十八位：^[1-9]\d{5}(18|19|([23]\d))\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$
				十五位：^[1-9]\d{5}\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}$
				总：
				(^[1-9]\d{5}(18|19|([23]\d))\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$)|(^[1-9]\d{5}\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}$)
			 */
		reg := regexp.MustCompile(`(^[1-9]\d{5}(18|19|([23]\d))\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$)|(^[1-9]\d{5}\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}$)`)

		return reg.MatchString(val)
	})
}
