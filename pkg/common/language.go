package common

var ValidLanguages = map[string]bool{
	"en":    true,
	"zh":    true,
	"zh_tw": true,
	"ru":    true,
}

var DefaultLanguage = "en"
var LanguageZh = "zh"
var LanguageZhTw = "zh_tw"
var LanguageRu = "ru"
var requestLanguage = DefaultLanguage

func SetRequestLanguage(language string) {
	if _, exists := ValidLanguages[language]; !exists {
		language = DefaultLanguage
	}
	requestLanguage = language
}

func GetRequestLanguage() string {
	return requestLanguage
}
