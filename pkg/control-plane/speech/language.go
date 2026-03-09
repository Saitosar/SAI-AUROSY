package speech

// Supported languages for the Speech Layer (uz, en, ru, az, ar).
const (
	LangUzbek     = "uz"
	LangEnglish   = "en"
	LangRussian   = "ru"
	LangAzerbaijani = "az"
	LangArabic    = "ar"
)

// SupportedLanguages is the list of language codes supported by the speech layer.
var SupportedLanguages = []string{LangUzbek, LangEnglish, LangRussian, LangAzerbaijani, LangArabic}

// IsSupported returns true if the language code is supported.
func IsSupported(lang string) bool {
	for _, l := range SupportedLanguages {
		if l == lang {
			return true
		}
	}
	return false
}

// LanguageTier indicates priority tier for language support.
type LanguageTier int

const (
	Tier1 LanguageTier = 1 // Uzbek, English, Russian
	Tier2 LanguageTier = 2 // Azerbaijani, Arabic
)

// Tier1Languages are the primary supported languages.
var Tier1Languages = []string{LangUzbek, LangEnglish, LangRussian}

// Tier2Languages are secondary supported languages.
var Tier2Languages = []string{LangAzerbaijani, LangArabic}

// GetTier returns the tier for a language, or 0 if unsupported.
func GetTier(lang string) LanguageTier {
	for _, l := range Tier1Languages {
		if l == lang {
			return Tier1
		}
	}
	for _, l := range Tier2Languages {
		if l == lang {
			return Tier2
		}
	}
	return 0
}

// LowConfidenceThreshold is the minimum confidence for accepting language detection.
// Below this, the system may ask the user to repeat or show a language selector.
const LowConfidenceThreshold = 0.7

// ShouldAskRepeat returns true if language confidence is too low.
func ShouldAskRepeat(confidence float64) bool {
	return confidence < LowConfidenceThreshold && confidence > 0
}
