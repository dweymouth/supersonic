package res

import "embed"

//go:embed translations
var Translations embed.FS

type TranslationInfo struct {
	Name                string
	DisplayName         string
	TranslationFileName string
}

var TranslationsInfo = []TranslationInfo{
	{Name: "de", DisplayName: "Deutsch", TranslationFileName: "de.json"},
	{Name: "en", DisplayName: "English", TranslationFileName: "en.json"},
	{Name: "es", DisplayName: "Español", TranslationFileName: "es.json"},
	{Name: "fr", DisplayName: "Français", TranslationFileName: "fr.json"},
	{Name: "it", DisplayName: "Italiano", TranslationFileName: "it.json"},
	{Name: "ja", DisplayName: "日本語", TranslationFileName: "ja.json"},
	{Name: "nl", DisplayName: "Nederlands", TranslationFileName: "nl.json"},
	{Name: "pl", DisplayName: "Polski", TranslationFileName: "pl.json"},
	{Name: "pt_BR", DisplayName: "Português (BR)", TranslationFileName: "pt_BR.json"},
	{Name: "ro", DisplayName: "Română", TranslationFileName: "ro.json"},
	{Name: "zhHans", DisplayName: "中文", TranslationFileName: "zhHans.json"},
	{Name: "zhHant", DisplayName: "中文 (trad.)", TranslationFileName: "zhHant.json"},
}
