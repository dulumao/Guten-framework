package validation

import (
	"github.com/go-playground/locales/en_US"
	"github.com/go-playground/locales/zh_Hans_CN"
	"github.com/go-playground/universal-translator"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/go-playground/validator.v9/translations/en"
	"gopkg.in/go-playground/validator.v9/translations/zh"
	"reflect"
	"regexp"
	"strings"
)

var (
	EnUs    = en_US.New()
	ZhHanCN = zh_Hans_CN.New()
)

type Validation struct {
	v *validator.Validate
}

var uni *ut.UniversalTranslator
var langPack = make(map[string]map[string]string)
var Validator *Validation

func New() *Validation {
	var validation = new(Validation)

	validation.v = validator.New()

	validation.v.SetTagName("valid")

	validation.v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]

		if name == "-" {
			return ""
		}

		// return "{{" + name + "}}"
		return name
	})

	validation.addValidations()
	validation.addLocale()

	Validator = validation

	return validation
}

// this is usually know or extracted from http 'Accept-Language' header
// also see uni.FindTranslator(...)
func (self *Validation) GetTranslator(locale string) (trans ut.Translator) {
	trans, found := uni.GetTranslator(locale)

	if found {
		return trans
	}

	return nil
}

func (self *Validation) addValidations() {
	self.v.RegisterValidation("phone", func(fl validator.FieldLevel) bool {
		field := fl.Field()
		param := fl.Param()

		_ = param

		reg := regexp.MustCompile(`^13[\d]{9}$|^14[5,7]{1}\d{8}$|^15[^4]{1}\d{8}$|^17[0,3,5,6,7,8]{1}\d{8}$|^18[\d]{9}$`)

		return reg.MatchString(field.String())
	})
	self.v.RegisterValidation("telephone", func(fl validator.FieldLevel) bool {
		// 国内座机电话号码："XXXX-XXXXXXX"、"XXXX-XXXXXXXX"、"XXX-XXXXXXX"、"XXX-XXXXXXXX"、"XXXXXXX"、"XXXXXXXX"
		field := fl.Field()
		param := fl.Param()

		_ = param

		reg := regexp.MustCompile(`^((\d{3,4})|\d{3,4}-)?\d{7,8}$`)

		return reg.MatchString(field.String())
	})
	self.v.RegisterValidation("qq", func(fl validator.FieldLevel) bool {
		// 腾讯QQ号，从10000开始
		field := fl.Field()
		param := fl.Param()

		_ = param

		reg := regexp.MustCompile(`^[1-9][0-9]{4,}$`)

		return reg.MatchString(field.String())
	})
	self.v.RegisterValidation("cn-postcode", func(fl validator.FieldLevel) bool {
		// 中国邮政编码
		field := fl.Field()
		param := fl.Param()

		_ = param

		reg := regexp.MustCompile(`^\d{6}$`)

		return reg.MatchString(field.String())
	})
	self.v.RegisterValidation("username", func(fl validator.FieldLevel) bool {
		// 通用帐号规则(字母开头，只能包含字母、数字和下划线，长度在6~18之间)
		field := fl.Field()
		param := fl.Param()

		_ = param

		reg := regexp.MustCompile(`^[a-zA-Z]{1}\w{5,17}$`)

		return reg.MatchString(field.String())
	})
	self.v.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		// 通用密码(任意可见字符，长度在6~18之间)
		field := fl.Field()
		param := fl.Param()

		_ = param

		reg := regexp.MustCompile(`^[\w\S]{6,18}$`)

		return reg.MatchString(field.String())
	})
	self.v.RegisterValidation("password2", func(fl validator.FieldLevel) bool {
		// 中等强度密码(在弱密码的基础上，必须包含大小写字母和数字)
		field := fl.Field()
		param := fl.Param()

		_ = param

		reg1 := regexp.MustCompile(`^[\w\S]{6,18}$`)
		reg2 := regexp.MustCompile(`[a-z]+`)
		reg3 := regexp.MustCompile(`[A-Z]+`)
		reg4 := regexp.MustCompile(`\d+`)

		return reg1.MatchString(field.String()) && reg2.MatchString(field.String()) && reg3.MatchString(field.String()) && reg4.MatchString(field.String())
	})
	self.v.RegisterValidation("cn-id-number", func(fl validator.FieldLevel) bool {
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
		field := fl.Field()
		param := fl.Param()

		_ = param

		reg := regexp.MustCompile(`(^[1-9]\d{5}(18|19|([23]\d))\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$)|(^[1-9]\d{5}\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}$)`)

		return reg.MatchString(field.String())
	})
}

func (self *Validation) addValidationsTrans(trans ut.Translator) {
	// Trans.Add("Age", "[Age]", true)
	// Trans.Add("Email", "[Email]", true)
	// Trans.Add("MobilePhone", "[Mobile Phone]", true)

	var translationFn validator.TranslationFunc = func(ut ut.Translator, fe validator.FieldError) string {
		t, err := ut.T(fe.Tag(), fe.Field(), fe.Param())
		if err != nil {
			return fe.(error).Error()
		}

		return t
	}

	var registerFn = func(key string, override bool) validator.RegisterTranslationsFunc {
		return func(ut ut.Translator) error {
			return ut.Add(key, langPack[ut.Locale()][key], override)
		}
	}

	self.v.RegisterTranslation("phone", trans, registerFn("phone", false), translationFn)
	self.v.RegisterTranslation("telephone", trans, registerFn("telephone", false), translationFn)
	self.v.RegisterTranslation("qq", trans, registerFn("qq", false), translationFn)
	self.v.RegisterTranslation("cn-postcode", trans, registerFn("cn-postcode", false), translationFn)
	self.v.RegisterTranslation("username", trans, registerFn("username", false), translationFn)
	self.v.RegisterTranslation("password", trans, registerFn("password", false), translationFn)
	self.v.RegisterTranslation("password2", trans, registerFn("password2", false), translationFn)
	self.v.RegisterTranslation("cn-id-number", trans, registerFn("cn-id-number", false), translationFn)
}

func (self *Validation) addLocale() {
	transEnUs, _ := uni.GetTranslator(EnUs.Locale())
	transZhHanCn, _ := uni.GetTranslator(ZhHanCN.Locale())

	en.RegisterDefaultTranslations(self.v, transEnUs)
	zh.RegisterDefaultTranslations(self.v, transZhHanCn)

	self.addValidationsTrans(transEnUs)
	self.addValidationsTrans(transZhHanCn)
}

func (self *Validation) Validate(i interface{}) error {
	return self.v.Struct(i)
}

func GetErrorFields(err error, trans ut.Translator) map[string]string {
	var errMsg = make(map[string]string)

	if trans == nil {
		return err.(validator.ValidationErrors).Translate(nil)
	}

	for _, err := range err.(validator.ValidationErrors) {

		/*
			fmt.Println("Namespace: " + err.Namespace())
			fmt.Println("Field: " + err.Field())
			fmt.Println("StructNamespace: " + err.StructNamespace()) // can differ when a custom TagNameFunc is registered or
			fmt.Println("StructField: " + err.StructField())         // by passing alt name to ReportError like below
			fmt.Println("Tag: " + err.Tag())
			fmt.Println("ActualTag: " + err.ActualTag())
			fmt.Println("Kind: ", err.Kind())
			fmt.Println("Type: ", err.Type())
			fmt.Println("Value: ", err.Value())
			fmt.Println("Param: " + err.Param())
			fmt.Println(err.Translate(transFr))
			fmt.Println()
		*/

		transFieldName, _ := trans.T(err.Field())
		errMsg[err.Field()] = strings.Replace(err.Translate(trans), err.Field(), transFieldName, -1)
	}

	return errMsg
}

func init() {
	uni = ut.New(EnUs, EnUs, ZhHanCN)

	langPack[EnUs.Locale()] = map[string]string{
		"phone":        "mobile phone number is wrong!",
		"telephone":    "telephone number is wrong!",
		"qq":           "tencent number is wrong!",
		"cn-postcode":  "CN zipcode is wrong!",
		"username":     "username does not match the rule!",
		"password":     "password does not match the rule!",
		"password2":    "password does not match the rule!",
		"cn-id-number": "ID-card number does not match the rule!",
	}

	langPack[ZhHanCN.Locale()] = map[string]string{
		"phone":        "手机号码错误!",
		"telephone":    "座机号码错误!",
		"qq":           "QQ号错误!",
		"cn-postcode":  "邮政编码错误!",
		"username":     "用户名不符合规则!",
		"password":     "密码不符合规则!",
		"password2":    "密码不符合规则!",
		"cn-id-number": "身份证号不符合规则!",
	}
}
