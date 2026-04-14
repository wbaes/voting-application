package i18n

// Language represents a supported UI language code.
type Language string

const (
	EN Language = "en"
	NL Language = "nl"
	FR Language = "fr"
	ES Language = "es"
)

// LangInfo holds display metadata for a language.
type LangInfo struct {
	Code string
	Name string
	Flag string
}

// All lists the supported languages for the UI selector.
var All = []LangInfo{
	{"en", "English", "🇬🇧"},
	{"nl", "Nederlands", "🇳🇱"},
	{"fr", "Français", "🇫🇷"},
	{"es", "Español", "🇪🇸"},
}

// Parse validates a language string and returns the corresponding Language,
// defaulting to EN for any unrecognised value.
func Parse(s string) Language {
	switch Language(s) {
	case NL, FR, ES:
		return Language(s)
	default:
		return EN
	}
}

// T returns a translation function for the given language. The returned
// function looks up a key and falls back to English if the key is missing.
func T(lang Language) func(string) string {
	m := translations[lang]
	if m == nil {
		m = translations[EN]
	}
	en := translations[EN]
	return func(key string) string {
		if v, ok := m[key]; ok {
			return v
		}
		if v, ok := en[key]; ok {
			return v
		}
		return key
	}
}

//nolint:misspell
var translations = map[Language]map[string]string{
	EN: {
		"subtitle":            "Vote for your favourite photo!",
		"already_voted_title": "Thank you for voting!",
		"already_voted_body":  "You voted for this photo. Your vote has been recorded.",
		"view_results":        "View Live Results →",
		"instruction":         "Tap a photo to select it, then confirm your vote below.",
		"almost_there":        "Almost there!",
		"you_selected":        "You selected:",
		"enter_draw":          "Enter the draw",
		"optional":            "(optional)",
		"enter_draw_desc":     "Leave your details to win your favourite photo!",
		"name_placeholder":    "Your name",
		"email_placeholder":   "Email address",
		"phone_placeholder":   "Phone number",
		"confirm_vote":        "Confirm Vote",
		"vote_recorded_title": "Vote recorded!",
		"vote_recorded_body":  "Thank you for participating.",
		"network_error":       "Network error. Please try again.",
		"thankyou_title":      "Thank you for voting!",
		"thankyou_body":       "Your vote has been recorded.",
		"live_results":        "Live Results",
		"total_votes":         "Total votes:",
		"votes":               "votes",
		"connecting":          "Connecting...",
		"ws_live":             "● Live",
		"ws_reconnecting":     "○ Reconnecting...",
	},
	NL: {
		"subtitle":            "Stem op uw favoriete foto!",
		"already_voted_title": "Bedankt voor uw stem!",
		"already_voted_body":  "U hebt op deze foto gestemd. Uw stem is geregistreerd.",
		"view_results":        "Bekijk live resultaten →",
		"instruction":         "Tik op een foto om te selecteren en bevestig uw stem.",
		"almost_there":        "Bijna klaar!",
		"you_selected":        "U koos:",
		"enter_draw":          "Doe mee aan de verloting",
		"optional":            "(optioneel)",
		"enter_draw_desc":     "Laat uw gegevens achter voor een kans op uw favoriete foto!",
		"name_placeholder":    "Uw naam",
		"email_placeholder":   "E-mailadres",
		"phone_placeholder":   "Telefoonnummer",
		"confirm_vote":        "Stem bevestigen",
		"vote_recorded_title": "Stem geregistreerd!",
		"vote_recorded_body":  "Bedankt voor uw deelname.",
		"network_error":       "Netwerkfout. Probeer het opnieuw.",
		"thankyou_title":      "Bedankt voor uw stem!",
		"thankyou_body":       "Uw stem is geregistreerd.",
		"live_results":        "Live resultaten",
		"total_votes":         "Totaal aantal stemmen:",
		"votes":               "stemmen",
		"connecting":          "Verbinden...",
		"ws_live":             "● Live",
		"ws_reconnecting":     "○ Opnieuw verbinden...",
	},
	FR: {
		"subtitle":            "Votez pour votre photo préférée\u00a0!",
		"already_voted_title": "Merci pour votre vote\u00a0!",
		"already_voted_body":  "Vous avez voté pour cette photo. Votre vote a été enregistré.",
		"view_results":        "Voir les résultats en direct →",
		"instruction":         "Touchez une photo pour la sélectionner, puis confirmez votre vote.",
		"almost_there":        "Presque\u00a0!",
		"you_selected":        "Vous avez choisi\u00a0:",
		"enter_draw":          "Participer au tirage",
		"optional":            "(optionnel)",
		"enter_draw_desc":     "Laissez vos coordonnées pour gagner votre photo préférée\u00a0!",
		"name_placeholder":    "Votre nom",
		"email_placeholder":   "Adresse e-mail",
		"phone_placeholder":   "Numéro de téléphone",
		"confirm_vote":        "Confirmer le vote",
		"vote_recorded_title": "Vote enregistré\u00a0!",
		"vote_recorded_body":  "Merci de votre participation.",
		"network_error":       "Erreur réseau. Veuillez réessayer.",
		"thankyou_title":      "Merci pour votre vote\u00a0!",
		"thankyou_body":       "Votre vote a été enregistré.",
		"live_results":        "Résultats en direct",
		"total_votes":         "Total des votes\u00a0:",
		"votes":               "votes",
		"connecting":          "Connexion...",
		"ws_live":             "● En direct",
		"ws_reconnecting":     "○ Reconnexion...",
	},
	ES: {
		"subtitle":            "¡Vota por tu foto favorita!",
		"already_voted_title": "¡Gracias por votar!",
		"already_voted_body":  "Votaste por esta foto. Tu voto ha sido registrado.",
		"view_results":        "Ver resultados en vivo →",
		"instruction":         "Toca una foto para seleccionarla y confirma tu voto.",
		"almost_there":        "¡Casi listo!",
		"you_selected":        "Seleccionaste:",
		"enter_draw":          "Participar en el sorteo",
		"optional":            "(opcional)",
		"enter_draw_desc":     "¡Deja tus datos para ganar tu foto favorita!",
		"name_placeholder":    "Tu nombre",
		"email_placeholder":   "Correo electrónico",
		"phone_placeholder":   "Número de teléfono",
		"confirm_vote":        "Confirmar voto",
		"vote_recorded_title": "¡Voto registrado!",
		"vote_recorded_body":  "Gracias por participar.",
		"network_error":       "Error de red. Por favor, inténtalo de nuevo.",
		"thankyou_title":      "¡Gracias por votar!",
		"thankyou_body":       "Tu voto ha sido registrado.",
		"live_results":        "Resultados en vivo",
		"total_votes":         "Total de votos:",
		"votes":               "votos",
		"connecting":          "Conectando...",
		"ws_live":             "● En vivo",
		"ws_reconnecting":     "○ Reconectando...",
	},
}
