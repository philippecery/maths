package i18n

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/goodsign/monday"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

var bundle *i18n.Bundle
var localizers map[string]*i18n.Localizer
var languages map[string]string

func init() {
	bundle = i18n.NewBundle(language.AmericanEnglish)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	localizers = make(map[string]*i18n.Localizer)
	loadMessages("i18n/messages.en-US.json", "en-US")
	loadMessages("i18n/messages.fr-FR.json", "fr-FR")
	initSupportedLanguages()
}

func loadMessages(path, lang string) {
	bundle.MustLoadMessageFile(path)
	localizers[lang] = i18n.NewLocalizer(bundle, lang)
}

// GetLocalizedMessage returns the message messageID in language lang
func GetLocalizedMessage(lang, messageID string, data ...interface{}) string {
	var pluralCount, templateData interface{}
	if len(data) > 0 {
		pluralCount = data[0]
		if len(data) > 1 {
			templateData = data[1]
		}
	}
	return localizers[lang].MustLocalize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		PluralCount:  pluralCount,
		TemplateData: templateData,
	})
}

func initSupportedLanguages() {
	languages = make(map[string]string)
	for _, languageTag := range bundle.LanguageTags() {
		log.Printf("Supported language: %s (%s)\n", display.Self.Name(languageTag), languageTag)
		languages[languageTag.String()] = GetLocalizedMessage(languageTag.String(), "language")
	}
}

// GetSupportedLanguages returns the list of supported languages, with their localized names
func GetSupportedLanguages() map[string]string {
	languagesCopy := make(map[string]string)
	for key, value := range languages {
		languagesCopy[key] = value
	}
	return languagesCopy
}

// GetSelectedLanguage returns the selected language retrieved from the cookie
func GetSelectedLanguage(r *http.Request) string {
	if cookie, err := r.Cookie("lang"); err == nil && languages[cookie.Value] != "" {
		return cookie.Value
	}
	return "en-US"
}

const dateFormat = "Monday 02 January 2006"
const dateTimeFormat = dateFormat + " @ 15:04:05 GMT"

// FormatDate returns the provided formatted and localized date.
func FormatDate(dateTime time.Time, locale string) string {
	return format(dateTime, locale, dateFormat)
}

// FormatDateTime returns the provided formatted and localized date and time.
func FormatDateTime(dateTime time.Time, locale string) string {
	return format(dateTime, locale, dateTimeFormat)
}

func format(dateTime time.Time, locale, format string) string {
	return monday.Format(dateTime, format, monday.Locale(strings.Replace(locale, "-", "_", 1)))
}
