package binder

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/labstack/echo"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type (
	// Binder is the default implementation of the Echo Binder interface.
	Binder struct {
		IgnoreError bool
	}

	// BindUnmarshaler is the interface used to wrap the UnmarshalParam method.
	BindUnmarshaler interface {
		// UnmarshalParam decodes and assigns a value from an form or query param.
		UnmarshalParam(param string) error
	}
)

func New(ignoreError ...bool) *Binder {
	var IgnoreError = false

	if len(ignoreError) > 0 {
		IgnoreError = ignoreError[0]
	}

	return &Binder{
		IgnoreError: IgnoreError,
	}
}

// Bind implements the `Binder#Bind` function.
func (b *Binder) Bind(i interface{}, c echo.Context) (err error) {
	req := c.Request()
	if req.ContentLength == 0 {
		if req.Method == http.MethodGet || req.Method == http.MethodDelete {
			if err = b.bindData(i, c.QueryParams(), "query"); err != nil {
				if b.IgnoreError {
					return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
				}
			}
			return
		}
		return echo.NewHTTPError(http.StatusBadRequest, "Request body can't be empty")
	}
	ctype := req.Header.Get(echo.HeaderContentType)
	switch {
	case strings.HasPrefix(ctype, echo.MIMEApplicationJSON):
		if err = json.NewDecoder(req.Body).Decode(i); err != nil {
			if ute, ok := err.(*json.UnmarshalTypeError); ok {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v", ute.Type, ute.Value, ute.Field, ute.Offset)).SetInternal(err)
			} else if se, ok := err.(*json.SyntaxError); ok {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Syntax error: offset=%v, error=%v", se.Offset, se.Error())).SetInternal(err)
			} else {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
	case strings.HasPrefix(ctype, echo.MIMEApplicationXML), strings.HasPrefix(ctype, echo.MIMETextXML):
		if err = xml.NewDecoder(req.Body).Decode(i); err != nil {
			if ute, ok := err.(*xml.UnsupportedTypeError); ok {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unsupported type error: type=%v, error=%v", ute.Type, ute.Error())).SetInternal(err)
			} else if se, ok := err.(*xml.SyntaxError); ok {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Syntax error: line=%v, error=%v", se.Line, se.Error())).SetInternal(err)
			} else {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
	case strings.HasPrefix(ctype, echo.MIMEApplicationForm), strings.HasPrefix(ctype, echo.MIMEMultipartForm):
		params, err := c.FormParams()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		if err = b.bindData(i, params, "form"); err != nil {
			if b.IgnoreError {
				return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
			}
		}
	default:
		return echo.ErrUnsupportedMediaType
	}
	return
}

func (b *Binder) bindData(ptr interface{}, data map[string][]string, tag string) error {
	typ := reflect.TypeOf(ptr).Elem()
	val := reflect.ValueOf(ptr).Elem()

	if typ.Kind() != reflect.Struct {
		return errors.New("binding element must be a struct")
	}

	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)

		if !structField.CanSet() {
			continue
		}

		structFieldKind := structField.Kind()
		inputFieldName := typeField.Tag.Get(tag)

		if inputFieldName == "" {
			inputFieldName = typeField.Name
			// If tag is nil, we inspect if the field is a struct.
			if _, ok := bindUnmarshaler(structField); !ok && structFieldKind == reflect.Struct {
				if err := b.bindData(structField.Addr().Interface(), data, tag); err != nil {
					return err
				}
				continue
			}
		}

		inputValue, exists := data[inputFieldName]
		if !exists {
			// Go json.Unmarshal supports case insensitive binding.  However the
			// url params are bound case sensitive which is inconsistent.  To
			// fix this we must check all of the map values in a
			// case-insensitive search.
			inputFieldName = strings.ToLower(inputFieldName)
			for k, v := range data {
				if strings.ToLower(k) == inputFieldName {
					inputValue = v
					exists = true
					break
				}
			}
		}

		if !exists {
			continue
		}

		// Call this first, in case we're dealing with an alias to an array type
		if ok, err := unmarshalField(typeField.Type.Kind(), inputValue[0], structField); ok {
			if err != nil {
				return err
			}
			continue
		}

		numElems := len(inputValue)

		if structFieldKind == reflect.Slice && numElems > 0 {
			sliceOf := structField.Type().Elem().Kind()
			slice := reflect.MakeSlice(structField.Type(), numElems, numElems)
			for j := 0; j < numElems; j++ {
				if err := setWithProperType(sliceOf, inputValue[j], slice.Index(j)); err != nil {
					return err
				}
			}
			val.Field(i).Set(slice)
		} else if err := setWithProperType(typeField.Type.Kind(), inputValue[0], structField); err != nil {
			return err

		}
	}
	return nil
}

func setWithProperType(valueKind reflect.Kind, val string, structField reflect.Value) error {
	// But also call it here, in case we're dealing with an array of BindUnmarshalers
	if ok, err := unmarshalField(valueKind, val, structField); ok {
		return err
	}

	var (
		_        = reflect.TypeOf(string(""))
		_        = reflect.TypeOf(map[string]interface{}{})
		timeType = reflect.TypeOf(time.Time{})
		_        = reflect.TypeOf(&time.Time{})
		urlType  = reflect.TypeOf(url.URL{})
	)

	switch valueKind {
	case reflect.Ptr:
		return setWithProperType(structField.Elem().Kind(), val, structField.Elem())
	case reflect.Int:
		return setIntField(val, 0, structField)
	case reflect.Int8:
		return setIntField(val, 8, structField)
	case reflect.Int16:
		return setIntField(val, 16, structField)
	case reflect.Int32:
		return setIntField(val, 32, structField)
	case reflect.Int64:
		return setIntField(val, 64, structField)
	case reflect.Uint:
		return setUintField(val, 0, structField)
	case reflect.Uint8:
		return setUintField(val, 8, structField)
	case reflect.Uint16:
		return setUintField(val, 16, structField)
	case reflect.Uint32:
		return setUintField(val, 32, structField)
	case reflect.Uint64:
		return setUintField(val, 64, structField)
	case reflect.Bool:
		return setBoolField(val, structField)
	case reflect.Float32:
		return setFloatField(val, 32, structField)
	case reflect.Float64:
		return setFloatField(val, 64, structField)
	case reflect.String:
		structField.SetString(val)
	case reflect.Struct:
		if structField.Type().ConvertibleTo(timeType) {
			return setTimeField(structField, val)
		} else if structField.Type().ConvertibleTo(urlType) {
			return setURLField(structField, val)
		}
	default:
		return errors.New("unknown type")
	}
	return nil
}

func unmarshalField(valueKind reflect.Kind, val string, field reflect.Value) (bool, error) {
	switch valueKind {
	case reflect.Ptr:
		return unmarshalFieldPtr(val, field)
	default:
		return unmarshalFieldNonPtr(val, field)
	}
}

// bindUnmarshaler attempts to unmarshal a reflect.Value into a BindUnmarshaler
func bindUnmarshaler(field reflect.Value) (BindUnmarshaler, bool) {
	ptr := reflect.New(field.Type())
	if ptr.CanInterface() {
		iface := ptr.Interface()
		if unmarshaler, ok := iface.(BindUnmarshaler); ok {
			return unmarshaler, ok
		}
	}
	return nil, false
}

func unmarshalFieldNonPtr(value string, field reflect.Value) (bool, error) {
	if unmarshaler, ok := bindUnmarshaler(field); ok {
		err := unmarshaler.UnmarshalParam(value)
		field.Set(reflect.ValueOf(unmarshaler).Elem())
		return true, err
	}
	return false, nil
}

func unmarshalFieldPtr(value string, field reflect.Value) (bool, error) {
	if field.IsNil() {
		// Initialize the pointer to a nil value
		field.Set(reflect.New(field.Type().Elem()))
	}
	return unmarshalFieldNonPtr(value, field.Elem())
}

func setIntField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	intVal, err := strconv.ParseInt(value, 10, bitSize)
	if err == nil {
		field.SetInt(intVal)
	}
	return err
}

func setUintField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0"
	}
	uintVal, err := strconv.ParseUint(value, 10, bitSize)
	if err == nil {
		field.SetUint(uintVal)
	}
	return err
}

func setBoolField(value string, field reflect.Value) error {
	if value == "" {
		value = "false"
	}
	boolVal, err := strconv.ParseBool(value)
	if err == nil {
		field.SetBool(boolVal)
	}
	return err
}

func setFloatField(value string, bitSize int, field reflect.Value) error {
	if value == "" {
		value = "0.0"
	}
	floatVal, err := strconv.ParseFloat(value, bitSize)
	if err == nil {
		field.SetFloat(floatVal)
	}
	return err
}

func setTimeField(v reflect.Value, s string) error {
	if s == "" {
		s = time.Time{}.String()
	}

	t := v.Type()

	p, err := parseTime(s)
	if err == nil {
		v.Set(reflect.ValueOf(p).Convert(v.Type()))
		return nil
	}

	return errors.New("cannot decode string `" + s + "` as " + t.String())
}

func setURLField(v reflect.Value, s string) error {
	t := v.Type()

	u, err := url.Parse(s)

	if err == nil {
		v.Set(reflect.ValueOf(*u).Convert(v.Type()))
		return nil
	}

	return errors.New("cannot decode string `" + s + "` as " + t.String())
}

func parseTime(datestr string) (time.Time, error) {
	state := stateStart

	firstSlash := 0

	// General strategy is to read rune by rune through the date looking for
	// certain hints of what type of date we are dealing with.
	// Hopefully we only need to read about 5 or 6 bytes before
	// we figure it out and then attempt a parse
iterRunes:
	for i := 0; i < len(datestr); i++ {
		r := rune(datestr[i])
		// r, bytesConsumed := utf8.DecodeRuneInString(datestr[ri:])
		// if bytesConsumed > 1 {
		// 	ri += (bytesConsumed - 1)
		// }

		switch state {
		case stateStart:
			if unicode.IsDigit(r) {
				state = stateDigit
			} else if unicode.IsLetter(r) {
				state = stateAlpha
			}
		case stateDigit: // starts digits
			if unicode.IsDigit(r) {
				continue
			} else if unicode.IsLetter(r) {
				state = stateDigitAlpha
				continue
			}
			switch r {
			case '-', '\u2212':
				state = stateDigitDash
			case '/':
				state = stateDigitSlash
				firstSlash = i
			}
		case stateDigitDash: // starts digit then dash 02-
			// 2006-01-02T15:04:05Z07:00
			// 2017-06-25T17:46:57.45706582-07:00
			// 2006-01-02T15:04:05.999999999Z07:00
			// 2006-01-02T15:04:05+0000
			// 2012-08-03 18:31:59.257000000
			// 2014-04-26 17:24:37.3186369
			// 2017-01-27 00:07:31.945167
			// 2016-03-14 00:00:00.000
			// 2014-05-11 08:20:13,787
			// 2017-07-19 03:21:51+00:00
			// 2006-01-02
			// 2013-04-01 22:43:22
			// 2014-04-26 05:24:37 PM
			// 2013-Feb-03
			switch {
			case r == ' ':
				state = stateDigitDashWs
			case r == 'T':
				state = stateDigitDashT
			default:
				if unicode.IsLetter(r) {
					state = stateDigitDashAlpha
					break iterRunes
				}
			}
		case stateDigitDashWs:
			// 2013-04-01 22:43:22
			// 2014-05-11 08:20:13,787
			// stateDigitDashWsWs
			//   2014-04-26 05:24:37 PM
			//   2014-12-16 06:20:00 UTC
			//   2015-02-18 00:12:00 +0000 UTC
			//   2006-01-02 15:04:05 -0700
			//   2006-01-02 15:04:05 -07:00
			// stateDigitDashWsOffset
			//   2017-07-19 03:21:51+00:00
			// stateDigitDashWsPeriod
			//   2014-04-26 17:24:37.3186369
			//   2017-01-27 00:07:31.945167
			//   2012-08-03 18:31:59.257000000
			//   2016-03-14 00:00:00.000
			//   stateDigitDashWsPeriodOffset
			//     2017-01-27 00:07:31.945167 +0000
			//     2016-03-14 00:00:00.000 +0000
			//     stateDigitDashWsPeriodOffsetAlpha
			//       2017-01-27 00:07:31.945167 +0000 UTC
			//       2016-03-14 00:00:00.000 +0000 UTC
			//   stateDigitDashWsPeriodAlpha
			//     2014-12-16 06:20:00.000 UTC
			switch r {
			case ',':
				if len(datestr) == len("2014-05-11 08:20:13,787") {
					// go doesn't seem to parse this one natively?   or did i miss it?
					t, err := parse("2006-01-02 03:04:05", datestr[:i])
					if err == nil {
						ms, err := strconv.Atoi(datestr[i+1:])
						if err == nil {
							return time.Unix(0, t.UnixNano()+int64(ms)*1e6), nil
						}
					}
					return t, err
				}
			case '-', '+':
				state = stateDigitDashWsOffset
			case '.':
				state = stateDigitDashWsPeriod
			case ' ':
				state = stateDigitDashWsWs
			}

		case stateDigitDashWsWs:
			// stateDigitDashWsWsAlpha
			//   2014-12-16 06:20:00 UTC
			//   stateDigitDashWsWsAMPMMaybe
			//     2014-04-26 05:24:37 PM
			// stateDigitDashWsWsOffset
			//   2006-01-02 15:04:05 -0700
			//   stateDigitDashWsWsOffsetColon
			//     2006-01-02 15:04:05 -07:00
			//     stateDigitDashWsWsOffsetColonAlpha
			//       2015-02-18 00:12:00 +00:00 UTC
			//   stateDigitDashWsWsOffsetAlpha
			//     2015-02-18 00:12:00 +0000 UTC
			switch r {
			case 'A', 'P':
				state = stateDigitDashWsWsAMPMMaybe
			case '+', '-':
				state = stateDigitDashWsWsOffset
			default:
				if unicode.IsLetter(r) {
					// 2014-12-16 06:20:00 UTC
					state = stateDigitDashWsWsAlpha
					break iterRunes
				}
			}

		case stateDigitDashWsWsAMPMMaybe:
			if r == 'M' {
				return parse("2006-01-02 03:04:05 PM", datestr)
			}
			state = stateDigitDashWsWsAlpha

		case stateDigitDashWsWsOffset:
			// stateDigitDashWsWsOffset
			//   2006-01-02 15:04:05 -0700
			//   stateDigitDashWsWsOffsetColon
			//     2006-01-02 15:04:05 -07:00
			//     stateDigitDashWsWsOffsetColonAlpha
			//       2015-02-18 00:12:00 +00:00 UTC
			//   stateDigitDashWsWsOffsetAlpha
			//     2015-02-18 00:12:00 +0000 UTC
			if r == ':' {
				state = stateDigitDashWsWsOffsetColon
			} else if unicode.IsLetter(r) {
				// 2015-02-18 00:12:00 +0000 UTC
				state = stateDigitDashWsWsOffsetAlpha
				break iterRunes
			}

		case stateDigitDashWsWsOffsetColon:
			// stateDigitDashWsWsOffsetColon
			//   2006-01-02 15:04:05 -07:00
			//   stateDigitDashWsWsOffsetColonAlpha
			//     2015-02-18 00:12:00 +00:00 UTC
			if unicode.IsLetter(r) {
				// 2015-02-18 00:12:00 +00:00 UTC
				state = stateDigitDashWsWsOffsetColonAlpha
				break iterRunes
			}

		case stateDigitDashWsPeriod:
			// 2014-04-26 17:24:37.3186369
			// 2017-01-27 00:07:31.945167
			// 2012-08-03 18:31:59.257000000
			// 2016-03-14 00:00:00.000
			// stateDigitDashWsPeriodOffset
			//   2017-01-27 00:07:31.945167 +0000
			//   2016-03-14 00:00:00.000 +0000
			//   stateDigitDashWsPeriodOffsetAlpha
			//     2017-01-27 00:07:31.945167 +0000 UTC
			//     2016-03-14 00:00:00.000 +0000 UTC
			// stateDigitDashWsPeriodAlpha
			//   2014-12-16 06:20:00.000 UTC
			if unicode.IsLetter(r) {
				// 2014-12-16 06:20:00.000 UTC
				state = stateDigitDashWsPeriodAlpha
				break iterRunes
			} else if r == '+' || r == '-' {
				state = stateDigitDashWsPeriodOffset
			}
		case stateDigitDashWsPeriodOffset:
			// 2017-01-27 00:07:31.945167 +0000
			// 2016-03-14 00:00:00.000 +0000
			// stateDigitDashWsPeriodOffsetAlpha
			//   2017-01-27 00:07:31.945167 +0000 UTC
			//   2016-03-14 00:00:00.000 +0000 UTC
			if unicode.IsLetter(r) {
				// 2014-12-16 06:20:00.000 UTC
				// 2017-01-27 00:07:31.945167 +0000 UTC
				// 2016-03-14 00:00:00.000 +0000 UTC
				state = stateDigitDashWsPeriodOffsetAlpha
				break iterRunes
			}
		case stateDigitDashT: // starts digit then dash 02-  then T
			// stateDigitDashT
			// 2006-01-02T15:04:05
			// stateDigitDashTZ
			// 2006-01-02T15:04:05.999999999Z
			// 2006-01-02T15:04:05.99999999Z
			// 2006-01-02T15:04:05.9999999Z
			// 2006-01-02T15:04:05.999999Z
			// 2006-01-02T15:04:05.99999Z
			// 2006-01-02T15:04:05.9999Z
			// 2006-01-02T15:04:05.999Z
			// 2006-01-02T15:04:05.99Z
			// 2009-08-12T22:15Z
			// stateDigitDashTZDigit
			// 2006-01-02T15:04:05.999999999Z07:00
			// 2006-01-02T15:04:05Z07:00
			// With another dash aka time-zone at end
			// stateDigitDashTOffset
			//   stateDigitDashTOffsetColon
			//     2017-06-25T17:46:57.45706582-07:00
			//     2017-06-25T17:46:57+04:00
			// 2006-01-02T15:04:05+0000
			switch r {
			case '-', '+':
				state = stateDigitDashTOffset
			case 'Z':
				state = stateDigitDashTZ
			}
		case stateDigitDashTZ:
			if unicode.IsDigit(r) {
				state = stateDigitDashTZDigit
			}
		case stateDigitDashTOffset:
			if r == ':' {
				state = stateDigitDashTOffsetColon
			}
		case stateDigitSlash: // starts digit then slash 02/
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 3/1/2014
			// 10/13/2014
			// 01/02/2006
			// 1/2/06
			if unicode.IsDigit(r) || r == '/' {
				continue
			}
			switch r {
			case ' ':
				state = stateDigitSlashWS
			}
		case stateDigitSlashWS: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			switch r {
			case ':':
				state = stateDigitSlashWSColon
			}
		case stateDigitSlashWSColon: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 3/1/2012 10:11:59 AM
			switch r {
			case ':':
				state = stateDigitSlashWSColonColon
			case 'A', 'P':
				state = stateDigitSlashWSColonAMPM
			}
		case stateDigitSlashWSColonColon: // starts digit then slash 02/ more digits/slashes then whitespace
			// 2014/07/10 06:55:38.156283
			// 03/19/2012 10:11:59
			// 04/2/2014 03:00:37
			// 3/1/2012 10:11:59
			// 4/8/2014 22:05
			// 3/1/2012 10:11:59 AM
			switch r {
			case 'A', 'P':
				state = stateDigitSlashWSColonColonAMPM
			}
		case stateDigitAlpha:
			// 12 Feb 2006, 19:17
			// 12 Feb 2006, 19:17:22
			switch {
			case len(datestr) == len("02 Jan 2006, 15:04"):
				return parse("02 Jan 2006, 15:04", datestr)
			case len(datestr) == len("02 Jan 2006, 15:04:05"):
				return parse("02 Jan 2006, 15:04:05", datestr)
			case len(datestr) == len("2006年01月02日"):
				return parse("2006年01月02日", datestr)
			case len(datestr) == len("2006年01月02日 15:04"):
				return parse("2006年01月02日 15:04", datestr)
			case strings.Contains(datestr, "ago"):
				state = stateHowLongAgo
			}
		case stateAlpha: // starts alpha
			// stateAlphaWS
			//  Mon Jan _2 15:04:05 2006
			//  Mon Jan _2 15:04:05 MST 2006
			//  Mon Jan 02 15:04:05 -0700 2006
			//  Mon Aug 10 15:44:11 UTC+0100 2015
			//  Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			//  stateAlphaWSDigitComma
			//    May 8, 2009 5:57:51 PM
			//
			// stateWeekdayComma
			//   Monday, 02-Jan-06 15:04:05 MST
			//   stateWeekdayCommaOffset
			//     Monday, 02 Jan 2006 15:04:05 -0700
			//     Monday, 02 Jan 2006 15:04:05 +0100
			// stateWeekdayAbbrevComma
			//   Mon, 02-Jan-06 15:04:05 MST
			//   Mon, 02 Jan 2006 15:04:05 MST
			//   stateWeekdayAbbrevCommaOffset
			//     Mon, 02 Jan 2006 15:04:05 -0700
			//     Thu, 13 Jul 2017 08:58:40 +0100
			//     stateWeekdayAbbrevCommaOffsetZone
			//       Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			switch {
			case unicode.IsLetter(r):
				continue
			case r == ' ':
				state = stateAlphaWS
			case r == ',':
				if i == 3 {
					state = stateWeekdayAbbrevComma
				} else {
					state = stateWeekdayComma
				}
			}
		case stateWeekdayComma: // Starts alpha then comma
			// Mon, 02-Jan-06 15:04:05 MST
			// Mon, 02 Jan 2006 15:04:05 MST
			// stateWeekdayCommaOffset
			//   Monday, 02 Jan 2006 15:04:05 -0700
			//   Monday, 02 Jan 2006 15:04:05 +0100
			switch {
			case r == '-':
				if i < 15 {
					return parse("Monday, 02-Jan-06 15:04:05 MST", datestr)
				}
				state = stateWeekdayCommaOffset
			case r == '+':
				state = stateWeekdayCommaOffset
			}
		case stateWeekdayAbbrevComma: // Starts alpha then comma
			// Mon, 02-Jan-06 15:04:05 MST
			// Mon, 02 Jan 2006 15:04:05 MST
			// stateWeekdayAbbrevCommaOffset
			//   Mon, 02 Jan 2006 15:04:05 -0700
			//   Thu, 13 Jul 2017 08:58:40 +0100
			//   stateWeekdayAbbrevCommaOffsetZone
			//     Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			switch {
			case r == '-':
				if i < 15 {
					return parse("Mon, 02-Jan-06 15:04:05 MST", datestr)
				}
				state = stateWeekdayAbbrevCommaOffset
			case r == '+':
				state = stateWeekdayAbbrevCommaOffset
			}

		case stateWeekdayAbbrevCommaOffset:
			// stateWeekdayAbbrevCommaOffset
			//   Mon, 02 Jan 2006 15:04:05 -0700
			//   Thu, 13 Jul 2017 08:58:40 +0100
			//   stateWeekdayAbbrevCommaOffsetZone
			//     Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
			if r == '(' {
				state = stateWeekdayAbbrevCommaOffsetZone
			}

		case stateAlphaWS: // Starts alpha then whitespace
			// Mon Jan _2 15:04:05 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			// Mon Aug 10 15:44:11 UTC+0100 2015
			switch {
			case unicode.IsLetter(r):
				state = stateAlphaWSAlpha
			case unicode.IsDigit(r):
				state = stateAlphaWSDigitComma
			}

		case stateAlphaWSDigitComma: // Starts Alpha, whitespace, digit, comma
			// May 8, 2009 5:57:51 PM
			// May 8, 2009
			if len(datestr) == len("May 8, 2009") {
				return parse("Jan 2, 2006", datestr)
			}
			return parse("Jan 2, 2006 3:04:05 PM", datestr)

		case stateAlphaWSAlpha: // Alpha, whitespace, alpha
			// Mon Jan _2 15:04:05 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if r == ':' {
				state = stateAlphaWSAlphaColon
			}
		case stateAlphaWSAlphaColon: // Alpha, whitespace, alpha, :
			// Mon Jan _2 15:04:05 2006
			// Mon Jan 02 15:04:05 -0700 2006
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if unicode.IsLetter(r) {
				state = stateAlphaWSAlphaColonAlpha
			} else if r == '-' || r == '+' {
				state = stateAlphaWSAlphaColonOffset
			}
		case stateAlphaWSAlphaColonAlpha: // Alpha, whitespace, alpha, :, alpha
			// Mon Jan _2 15:04:05 MST 2006
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if r == '+' {
				state = stateAlphaWSAlphaColonAlphaOffset
			}
		case stateAlphaWSAlphaColonAlphaOffset: // Alpha, whitespace, alpha, : , alpha, offset, ?
			// Mon Aug 10 15:44:11 UTC+0100 2015
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			if unicode.IsLetter(r) {
				state = stateAlphaWSAlphaColonAlphaOffsetAlpha
			}
		default:
			break iterRunes
		}
	}

	switch state {
	case stateDigit:
		// unixy timestamps ish
		//  1499979655583057426  nanoseconds
		//  1499979795437000     micro-seconds
		//  1499979795437        milliseconds
		//  1384216367189
		//  1332151919           seconds
		//  20140601             yyyymmdd
		//  2014                 yyyy
		t := time.Time{}
		if len(datestr) > len("1499979795437000") {
			if nanoSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				t = time.Unix(0, nanoSecs)
			}
		} else if len(datestr) > len("1499979795437") {
			if microSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				t = time.Unix(0, microSecs*1000)
			}
		} else if len(datestr) > len("1332151919") {
			if miliSecs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				t = time.Unix(0, miliSecs*1000*1000)
			}
		} else if len(datestr) == len("20140601") {
			return parse("20060102", datestr)
		} else if len(datestr) == len("2014") {
			return parse("2006", datestr)
		}
		if t.IsZero() {
			if secs, err := strconv.ParseInt(datestr, 10, 64); err == nil {
				if secs < 0 {
					// Now, for unix-seconds we aren't going to guess a lot
					// nothing before unix-epoch
				} else {
					t = time.Unix(secs, 0)
				}
			}
		}
		if !t.IsZero() {
			return t, nil
		}

	case stateDigitDash: // starts digit then dash 02-
		// 2006-01-02
		// 2006-01
		if len(datestr) == len("2014-04-26") {
			return parse("2006-01-02", datestr)
		} else if len(datestr) == len("2014-04") {
			return parse("2006-01", datestr)
		}
	case stateDigitDashAlpha:
		// 2013-Feb-03
		return parse("2006-Jan-02", datestr)

	case stateDigitDashTOffset:
		// 2006-01-02T15:04:05+0000
		return parse("2006-01-02T15:04:05-0700", datestr)

	case stateDigitDashTOffsetColon:
		// With another +/- time-zone at end
		// 2006-01-02T15:04:05.999999999+07:00
		// 2006-01-02T15:04:05.999999999-07:00
		// 2006-01-02T15:04:05.999999+07:00
		// 2006-01-02T15:04:05.999999-07:00
		// 2006-01-02T15:04:05.999+07:00
		// 2006-01-02T15:04:05.999-07:00
		// 2006-01-02T15:04:05+07:00
		// 2006-01-02T15:04:05-07:00
		return parse("2006-01-02T15:04:05-07:00", datestr)

	case stateDigitDashT: // starts digit then dash 02-  then T
		// 2006-01-02T15:04:05.999999
		// 2006-01-02T15:04:05.999999
		return parse("2006-01-02T15:04:05", datestr)

	case stateDigitDashTZDigit:
		// With a time-zone at end after Z
		// 2006-01-02T15:04:05.999999999Z07:00
		// 2006-01-02T15:04:05Z07:00
		// RFC3339     = "2006-01-02T15:04:05Z07:00"
		// RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
		return time.Time{}, fmt.Errorf("RFC339 Dates may not contain both Z & Offset for %q see https://github.com/golang/go/issues/5294", datestr)

	case stateDigitDashTZ: // starts digit then dash 02-  then T Then Z
		// 2006-01-02T15:04:05.999999999Z
		// 2006-01-02T15:04:05.99999999Z
		// 2006-01-02T15:04:05.9999999Z
		// 2006-01-02T15:04:05.999999Z
		// 2006-01-02T15:04:05.99999Z
		// 2006-01-02T15:04:05.9999Z
		// 2006-01-02T15:04:05.999Z
		// 2006-01-02T15:04:05.99Z
		// 2009-08-12T22:15Z  -- No seconds/milliseconds
		switch len(datestr) {
		case len("2009-08-12T22:15Z"):
			return parse("2006-01-02T15:04Z", datestr)
		default:
			return parse("2006-01-02T15:04:05Z", datestr)
		}
	case stateDigitDashWs: // starts digit then dash 02-  then whitespace   1 << 2  << 5 + 3
		// 2013-04-01 22:43:22
		return parse("2006-01-02 15:04:05", datestr)

	case stateDigitDashWsWsOffset:
		// 2006-01-02 15:04:05 -0700
		return parse("2006-01-02 15:04:05 -0700", datestr)

	case stateDigitDashWsWsOffsetColon:
		// 2006-01-02 15:04:05 -07:00
		return parse("2006-01-02 15:04:05 -07:00", datestr)

	case stateDigitDashWsWsOffsetAlpha:
		// 2015-02-18 00:12:00 +0000 UTC
		t, err := parse("2006-01-02 15:04:05 -0700 UTC", datestr)
		if err == nil {
			return t, nil
		}
		return parse("2006-01-02 15:04:05 +0000 GMT", datestr)

	case stateDigitDashWsWsOffsetColonAlpha:
		// 2015-02-18 00:12:00 +00:00 UTC
		return parse("2006-01-02 15:04:05 -07:00 UTC", datestr)

	case stateDigitDashWsOffset:
		// 2017-07-19 03:21:51+00:00
		return parse("2006-01-02 15:04:05-07:00", datestr)

	case stateDigitDashWsWsAlpha:
		// 2014-12-16 06:20:00 UTC
		t, err := parse("2006-01-02 15:04:05 UTC", datestr)
		if err == nil {
			return t, nil
		}
		t, err = parse("2006-01-02 15:04:05 GMT", datestr)
		if err == nil {
			return t, nil
		}
		if len(datestr) > len("2006-01-02 03:04:05") {
			t, err = parse("2006-01-02 03:04:05", datestr[:len("2006-01-02 03:04:05")])
			if err == nil {
				return t, nil
			}
		}

	case stateDigitDashWsPeriod:
		// 2012-08-03 18:31:59.257000000
		// 2014-04-26 17:24:37.3186369
		// 2017-01-27 00:07:31.945167
		// 2016-03-14 00:00:00.000
		return parse("2006-01-02 15:04:05", datestr)

	case stateDigitDashWsPeriodAlpha:
		// 2012-08-03 18:31:59.257000000 UTC
		// 2014-04-26 17:24:37.3186369 UTC
		// 2017-01-27 00:07:31.945167 UTC
		// 2016-03-14 00:00:00.000 UTC
		return parse("2006-01-02 15:04:05 UTC", datestr)

	case stateDigitDashWsPeriodOffset:
		// 2012-08-03 18:31:59.257000000 +0000
		// 2014-04-26 17:24:37.3186369 +0000
		// 2017-01-27 00:07:31.945167 +0000
		// 2016-03-14 00:00:00.000 +0000
		return parse("2006-01-02 15:04:05 -0700", datestr)

	case stateDigitDashWsPeriodOffsetAlpha:
		// 2012-08-03 18:31:59.257000000 +0000 UTC
		// 2014-04-26 17:24:37.3186369 +0000 UTC
		// 2017-01-27 00:07:31.945167 +0000 UTC
		// 2016-03-14 00:00:00.000 +0000 UTC
		return parse("2006-01-02 15:04:05 -0700 UTC", datestr)

	case stateAlphaWSAlphaColon:
		// Mon Jan _2 15:04:05 2006
		return parse(time.ANSIC, datestr)

	case stateAlphaWSAlphaColonOffset:
		// Mon Jan 02 15:04:05 -0700 2006
		return parse(time.RubyDate, datestr)

	case stateAlphaWSAlphaColonAlpha:
		// Mon Jan _2 15:04:05 MST 2006
		return parse(time.UnixDate, datestr)

	case stateAlphaWSAlphaColonAlphaOffset:
		// Mon Aug 10 15:44:11 UTC+0100 2015
		return parse("Mon Jan 02 15:04:05 MST-0700 2006", datestr)

	case stateAlphaWSAlphaColonAlphaOffsetAlpha:
		// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
		if len(datestr) > len("Mon Jan 02 2006 15:04:05 MST-0700") {
			// What effing time stamp is this?
			// Fri Jul 03 2015 18:04:07 GMT+0100 (GMT Daylight Time)
			dateTmp := datestr[:33]
			return parse("Mon Jan 02 2006 15:04:05 MST-0700", dateTmp)
		}
	case stateDigitSlash: // starts digit then slash 02/ (but nothing else)
		// 3/1/2014
		// 10/13/2014
		// 01/02/2006
		// 2014/10/13
		if firstSlash == 4 {
			if len(datestr) == len("2006/01/02") {
				return parse("2006/01/02", datestr)
			}
			return parse("2006/1/2", datestr)
		}
		for _, parseFormat := range shortDates {
			if t, err := parse(parseFormat, datestr); err == nil {
				return t, nil
			}
		}

	case stateDigitSlashWSColon: // starts digit then slash 02/ more digits/slashes then whitespace
		// 4/8/2014 22:05
		// 04/08/2014 22:05
		// 2014/4/8 22:05
		// 2014/04/08 22:05

		if firstSlash == 4 {
			for _, layout := range []string{"2006/01/02 15:04", "2006/1/2 15:04", "2006/01/2 15:04", "2006/1/02 15:04"} {
				if t, err := parse(layout, datestr); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 15:04", "01/2/2006 15:04", "1/02/2006 15:04", "1/2/2006 15:04"} {
				if t, err := parse(layout, datestr); err == nil {
					return t, nil
				}
			}
		}

	case stateDigitSlashWSColonAMPM: // starts digit then slash 02/ more digits/slashes then whitespace
		// 4/8/2014 22:05 PM
		// 04/08/2014 22:05 PM
		// 04/08/2014 1:05 PM
		// 2014/4/8 22:05 PM
		// 2014/04/08 22:05 PM

		if firstSlash == 4 {
			for _, layout := range []string{"2006/01/02 03:04 PM", "2006/01/2 03:04 PM", "2006/1/02 03:04 PM", "2006/1/2 03:04 PM",
				"2006/01/02 3:04 PM", "2006/01/2 3:04 PM", "2006/1/02 3:04 PM", "2006/1/2 3:04 PM"} {
				if t, err := parse(layout, datestr); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 03:04 PM", "01/2/2006 03:04 PM", "1/02/2006 03:04 PM", "1/2/2006 03:04 PM",
				"01/02/2006 3:04 PM", "01/2/2006 3:04 PM", "1/02/2006 3:04 PM", "1/2/2006 3:04 PM"} {
				if t, err := parse(layout, datestr); err == nil {
					return t, nil
				}

			}
		}

	case stateDigitSlashWSColonColon: // starts digit then slash 02/ more digits/slashes then whitespace double colons
		// 2014/07/10 06:55:38.156283
		// 03/19/2012 10:11:59
		// 3/1/2012 10:11:59
		// 03/1/2012 10:11:59
		// 3/01/2012 10:11:59
		if firstSlash == 4 {
			for _, layout := range []string{"2006/01/02 15:04:05", "2006/1/02 15:04:05", "2006/01/2 15:04:05", "2006/1/2 15:04:05"} {
				if t, err := parse(layout, datestr); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 15:04:05", "1/02/2006 15:04:05", "01/2/2006 15:04:05", "1/2/2006 15:04:05"} {
				if t, err := parse(layout, datestr); err == nil {
					return t, nil
				}
			}
		}

	case stateDigitSlashWSColonColonAMPM: // starts digit then slash 02/ more digits/slashes then whitespace double colons
		// 2014/07/10 06:55:38.156283 PM
		// 03/19/2012 10:11:59 PM
		// 3/1/2012 10:11:59 PM
		// 03/1/2012 10:11:59 PM
		// 3/01/2012 10:11:59 PM

		if firstSlash == 4 {
			for _, layout := range []string{"2006/01/02 03:04:05 PM", "2006/1/02 03:04:05 PM", "2006/01/2 03:04:05 PM", "2006/1/2 03:04:05 PM",
				"2006/01/02 3:04:05 PM", "2006/1/02 3:04:05 PM", "2006/01/2 3:04:05 PM", "2006/1/2 3:04:05 PM"} {
				if t, err := parse(layout, datestr); err == nil {
					return t, nil
				}
			}
		} else {
			for _, layout := range []string{"01/02/2006 03:04:05 PM", "1/02/2006 03:04:05 PM", "01/2/2006 03:04:05 PM", "1/2/2006 03:04:05 PM"} {
				if t, err := parse(layout, datestr); err == nil {
					return t, nil
				}
			}
		}

	case stateWeekdayCommaOffset:
		// Monday, 02 Jan 2006 15:04:05 -0700
		// Monday, 02 Jan 2006 15:04:05 +0100
		return parse("Monday, 02 Jan 2006 15:04:05 -0700", datestr)
	case stateWeekdayAbbrevComma: // Starts alpha then comma
		// Mon, 02-Jan-06 15:04:05 MST
		// Mon, 02 Jan 2006 15:04:05 MST
		return parse("Mon, 02 Jan 2006 15:04:05 MST", datestr)
	case stateWeekdayAbbrevCommaOffset:
		// Mon, 02 Jan 2006 15:04:05 -0700
		// Thu, 13 Jul 2017 08:58:40 +0100
		// RFC1123Z    = "Mon, 02 Jan 2006 15:04:05 -0700" // RFC1123 with numeric zone
		return parse("Mon, 02 Jan 2006 15:04:05 -0700", datestr)
	case stateWeekdayAbbrevCommaOffsetZone:
		// Tue, 11 Jul 2017 16:28:13 +0200 (CEST)
		return parse("Mon, 02 Jan 2006 15:04:05 -0700 (CEST)", datestr)
	case stateHowLongAgo:
		// 1 minutes ago
		// 1 hours ago
		// 1 day ago
		switch len(datestr) {
		case len("1 minutes ago"), len("10 minutes ago"), len("100 minutes ago"):
			return agoTime(datestr, time.Minute)
		case len("1 hours ago"), len("10 hours ago"):
			return agoTime(datestr, time.Hour)
		case len("1 day ago"), len("10 day ago"):
			return agoTime(datestr, Day)
		}
	}

	return time.Time{}, fmt.Errorf("Could not find date format for %s", datestr)
}

func agoTime(datestr string, d time.Duration) (time.Time, error) {
	dstrs := strings.Split(datestr, " ")
	m, err := strconv.Atoi(dstrs[0])
	if err != nil {
		return time.Time{}, err
	}
	return time.Now().Add(-d * time.Duration(m)), nil
}

func parse(layout, datestr string) (time.Time, error) {
	return time.Parse(layout, datestr)
}

type dateState int

const (
	stateStart dateState = iota
	stateDigit
	stateDigitDash
	stateDigitDashAlpha
	stateDigitDashWs
	stateDigitDashWsWs
	stateDigitDashWsWsAMPMMaybe
	stateDigitDashWsWsOffset
	stateDigitDashWsWsOffsetAlpha
	stateDigitDashWsWsOffsetColonAlpha
	stateDigitDashWsWsOffsetColon
	stateDigitDashWsOffset
	stateDigitDashWsWsAlpha
	stateDigitDashWsPeriod
	stateDigitDashWsPeriodAlpha
	stateDigitDashWsPeriodOffset
	stateDigitDashWsPeriodOffsetAlpha
	stateDigitDashT
	stateDigitDashTZ
	stateDigitDashTZDigit
	stateDigitDashTOffset
	stateDigitDashTOffsetColon
	stateDigitSlash
	stateDigitSlashWS
	stateDigitSlashWSColon
	stateDigitSlashWSColonAMPM
	stateDigitSlashWSColonColon
	stateDigitSlashWSColonColonAMPM
	stateDigitAlpha
	stateAlpha
	stateAlphaWS
	stateAlphaWSDigitComma
	stateAlphaWSAlpha
	stateAlphaWSAlphaColon
	stateAlphaWSAlphaColonOffset
	stateAlphaWSAlphaColonAlpha
	stateAlphaWSAlphaColonAlphaOffset
	stateAlphaWSAlphaColonAlphaOffsetAlpha
	stateWeekdayComma
	stateWeekdayCommaOffset
	stateWeekdayAbbrevComma
	stateWeekdayAbbrevCommaOffset
	stateWeekdayAbbrevCommaOffsetZone
	stateHowLongAgo
)

const (
	Day = time.Hour * 24
)

var (
	shortDates = []string{"01/02/2006", "1/2/2006", "06/01/02", "01/02/06", "1/2/06"}
)
