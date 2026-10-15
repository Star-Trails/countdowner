package main

import "countdowner/locales"

// Keep the user's requested language order, independent of the current locale.
var languageCodes = []string{"en", "zh-Hant", "zh-Hans", "ja", "ko"}
var languageNames = []string{"English", "正體中文", "简体中文", "日本語", "한국어"}

func (a *application) text() locales.Messages {
	return locales.In(a.config.Language)
}

func (a *application) statusText(value string) string {
	text := a.text()
	switch value {
	case "Saved":
		return text.Saved()
	case "Enter a title":
		return text.EnterATitle()
	case "Enter a valid local date or time":
		return text.InvalidDateTime()
	case "Configuration is read-only; the original file is preserved":
		return text.ReadOnlyConfig()
	case "Desktop connected":
		return text.DesktopConnected()
	case "Waiting for Explorer desktop":
		return text.WaitingForDesktop()
	default:
		return value
	}
}
