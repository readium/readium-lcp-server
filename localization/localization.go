package localization

import (
  "path"
	"github.com/nicksnyder/go-i18n/i18n"
	"github.com/readium/readium-lcp-server/config"
)

//InitTranslations loads files with translation according to array in config file
//need to run in main.go in server
//err!=nil  means that one of them can't be opened
func InitTranslations() error {
	acceptableLanguages := config.Config.Localization.Languages
	localizationPath := config.Config.Localization.Folder

	var err error
	for _, value := range acceptableLanguages {
		err = i18n.LoadTranslationFile(path.Join(localizationPath, value + ".json"))
	}

	return err
}

//LocalizeMessage translates messages
//acceptLanguage - Accept-Languages from request header (r.Header.Get("Accept-Language"))
func LocalizeMessage(acceptLanguage string, message *string, key string) {
	defaultLanguage := config.Config.Localization.DefaultLanguage

	T, _ := i18n.Tfunc(acceptLanguage, defaultLanguage)
	*message = T(key)
}
