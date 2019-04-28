package i18n

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"github.com/labstack/echo"
	"strconv"
	"regexp"
	"errors"
	"net/http"
)

type LocaleType struct {
	Language string
	Country  string
	Score    float64
}

type LocaleTypes []*LocaleType

var (
	lcRegex = regexp.MustCompile("[a-zA-Z]+")
	qRegex  = regexp.MustCompile("[0-9]+.?[0-9]*")
)

func New() {
	filepath.Walk("resources/locales", func(path string, info os.FileInfo, err error) error {
		if strings.Contains(path, ".ini") {
			SetMessage(strings.Split(filepath.Base(path), ".")[0], path)
		}

		return nil
	})

	if files, err := ioutil.ReadDir("plugins"); err != nil {
		panic(err)
	} else {
		for _, f := range files {
			if f.IsDir() {
				pluginName := f.Name()
				pluginPath := "plugins" + string(os.PathSeparator) + pluginName + string(os.PathSeparator) + "resources/locales"

				filepath.Walk(pluginPath, func(path string, info os.FileInfo, err error) error {
					if strings.Contains(path, ".ini") {
						SetMessage(pluginName+"::"+strings.Split(filepath.Base(path), ".")[0], path)
					}

					return nil
				})
			}
		}
	}
}

func Default(context echo.Context) *Locale {
	var localeString = ""
	var browserLang = context.FormValue("lang")

	if browserLang != "" {
		context.SetCookie(&http.Cookie{
			Name:   "lang",
			Value:  browserLang,
			Path:   "/",
			MaxAge: 31536000,
			// MaxAge: -1 delete
		})

		localeString = browserLang
	} else {
		if lang, err := context.Cookie("lang"); err != nil {

			if acceptLanguage, err := NewLocale(context.Request().Header.Get("Accept-Language")); err != nil {
				// localeString = vars.Kernel.Config.Server.FallbackLocale
				localeString = "en"
			} else {
				localeString = acceptLanguage.Language + "-" + acceptLanguage.Country
			}

		} else {
			localeString = lang.Value
		}
	}

	return &Locale{
		Lang: localeString,
	}
}

// NewLocale delivers a reference to a Locale for an individual
// locale definition in an Accept-Language header.
//
// Example definitions:
//
//   en-US
//   en_GB
//   en;q=0.8
func NewLocale(l string) (*LocaleType, error) {
	// Break l into language / country and q
	parts := strings.Split(l, ";")

	// Grab language and country, with country being optional
	lc := lcRegex.FindAllStringSubmatch(parts[0], -1)

	if len(lc) == 0 {
		return nil, errors.New("No language or country provided")
	}

	// Initialize locale with language / country
	locale := &LocaleType{Language: lc[0][0]}
	locale.Language = lc[0][0]

	if len(lc) > 1 {
		locale.Country = lc[1][0]
	}

	// Determine if score is specified
	if len(parts) > 1 {
		score := qRegex.FindAllStringSubmatch(parts[1], -1)
		if len(score) > 0 {
			locale.Score, _ = strconv.ParseFloat(score[0][0], 64)
		}
	}

	// Default score
	if locale.Score == 0 {
		locale.Score = 1.0
	}

	return locale, nil
}

// Read receives a complete Accept-Language header and delivers a reference
// to a Locales, containing references to a *Locale for each definition in
// the Accept-Language header.
func Read(header string) LocaleTypes {
	// Initialize Locales
	locales := LocaleTypes{}

	// Split out individual definitions
	pieces := strings.Split(header, ",")

	// Add Locale for each definition
	for _, l := range pieces {
		if locale, err := NewLocale(l); err == nil {
			locales = append(locales, locale)
		}
	}

	return locales
}

// Best delivers the locale with the highest quality score
func (ls LocaleTypes) Best() *LocaleType {
	best := (*LocaleType)(nil)
	score := 0.0

	for _, l := range ls {
		if best == nil || l.Score > score {
			best = l
			score = l.Score
		}
	}

	return best
}

func (l *LocaleType) String() string {
	s := l.Language

	if l.Country != "" {
		s = s + "_" + l.Country
	}

	return s
}
