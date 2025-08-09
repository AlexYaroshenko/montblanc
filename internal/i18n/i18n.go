package i18n

import (
	"net/http"
	"strings"
)

// Supported languages: English (en), German (de), French (fr), Spanish (es), Italian (it)
var supported = map[string]map[string]string{
	"en": {
        "title":              "Refuge Availability",
        "last_updated":       "Last updated",
        "places":             "places",
        "available":          "Available",
        "full":               "Full",
        "subscribe":          "Subscribe",
        "chat_id":            "Telegram Chat ID",
        "language":           "Language",
        "refuge":             "Refuge",
        "date_from":          "From date",
        "date_to":            "To date",
        "submit":             "Submit",
        "success":            "Subscription saved",
        "hero_title":         "Free spots in Mont Blanc refuges — in real time",
        "hero_subtitle":      "No more daily checks. We'll notify you when spots appear.",
        "cta_check":          "Check availability",
        "cta_subscribe":      "Subscribe to alerts",
        "how_it_works_title": "How it works",
        "step1":              "We monitor official refuge sites 24/7",
        "step2":              "We show available dates and spots",
        "step3":              "We notify you via Telegram",
        "demo_title":         "Live availability",
        "refuges_title":      "Covered refuges",
        "sample_free_spots":  "Free spots",
        "sample_in":          "in",
        "try":                "Try",
        "subscribe_hint":     "Recommended: subscribe via Telegram in one click — press the button above and send /start. If you already know your Chat ID, you can fill the form below.",
        "chat_id_hint":       "Don't know your Chat ID?",
        "chat_id_how":        "Open the bot and send /id",
	},
	"de": {
        "title":              "Hüttenverfügbarkeit",
        "last_updated":       "Zuletzt aktualisiert",
        "places":             "Plätze",
        "available":          "Verfügbar",
        "full":               "Ausgebucht",
        "subscribe":          "Abonnieren",
        "chat_id":            "Telegram Chat-ID",
        "language":           "Sprache",
        "refuge":             "Hütte",
        "date_from":          "Von Datum",
        "date_to":            "Bis Datum",
        "submit":             "Senden",
        "success":            "Abonnement gespeichert",
        "hero_title":         "Freie Plätze in Mont-Blanc-Hütten – in Echtzeit",
        "hero_subtitle":      "Keine täglichen Checks mehr. Wir benachrichtigen Sie, wenn Plätze frei werden.",
        "cta_check":          "Verfügbarkeit prüfen",
        "cta_subscribe":      "Benachrichtigungen abonnieren",
        "how_it_works_title": "So funktioniert es",
        "step1":              "Wir überwachen die offiziellen Hüttenseiten rund um die Uhr",
        "step2":              "Wir zeigen verfügbare Daten und Plätze",
        "step3":              "Wir benachrichtigen Sie per Telegram",
        "demo_title":         "Live-Verfügbarkeit",
        "refuges_title":      "Abgedeckte Hütten",
        "sample_free_spots":  "Freie Plätze",
        "sample_in":          "in",
        "try":                "Ausprobieren",
        "subscribe_hint":     "Empfehlung: Abonniere via Telegram mit einem Klick – Button oben und /start senden. Wenn du deine Chat-ID kennst, fülle das Formular unten aus.",
        "chat_id_hint":       "Kennst du deine Chat-ID nicht?",
        "chat_id_how":        "Öffne den Bot und sende /id",
	},
	"fr": {
        "title":              "Disponibilité des refuges",
        "last_updated":       "Dernière mise à jour",
        "places":             "places",
        "available":          "Disponible",
        "full":               "Complet",
        "subscribe":          "S'abonner",
        "chat_id":            "ID de chat Telegram",
        "language":           "Langue",
        "refuge":             "Refuge",
        "date_from":          "Date de début",
        "date_to":            "Date de fin",
        "submit":             "Envoyer",
        "success":            "Abonnement enregistré",
        "hero_title":         "Places libres dans les refuges du Mont-Blanc — en temps réel",
        "hero_subtitle":      "Fini les vérifications quotidiennes. Nous vous avertissons dès qu'une place se libère.",
        "cta_check":          "Vérifier la disponibilité",
        "cta_subscribe":      "S'abonner aux alertes",
        "how_it_works_title": "Comment ça marche",
        "step1":              "Nous surveillons les sites officiels des refuges 24/7",
        "step2":              "Nous affichons les dates et places disponibles",
        "step3":              "Nous vous avertissons sur Telegram",
        "demo_title":         "Disponibilité en direct",
        "refuges_title":      "Refuges couverts",
        "sample_free_spots":  "Places libres",
        "sample_in":          "à",
        "try":                "Essayer",
        "subscribe_hint":     "Recommandé : inscrivez-vous via Telegram en un clic — bouton ci-dessus puis /start. Si vous connaissez votre Chat ID, vous pouvez remplir le formulaire ci-dessous.",
        "chat_id_hint":       "Vous ne connaissez pas votre Chat ID ?",
        "chat_id_how":        "Ouvrez le bot et envoyez /id",
	},
	"es": {
        "title":              "Disponibilidad de refugios",
        "last_updated":       "Última actualización",
        "places":             "plazas",
        "available":          "Disponible",
        "full":               "Completo",
        "subscribe":          "Suscribirse",
        "chat_id":            "ID de chat de Telegram",
        "language":           "Idioma",
        "refuge":             "Refugio",
        "date_from":          "Fecha de inicio",
        "date_to":            "Fecha de fin",
        "submit":             "Enviar",
        "success":            "Suscripción guardada",
        "hero_title":         "Plazas libres en refugios del Mont Blanc — en tiempo real",
        "hero_subtitle":      "No más comprobaciones diarias. Te avisamos cuando haya plazas.",
        "cta_check":          "Comprobar disponibilidad",
        "cta_subscribe":      "Suscribirse a alertas",
        "how_it_works_title": "Cómo funciona",
        "step1":              "Monitorizamos los sitios oficiales 24/7",
        "step2":              "Mostramos fechas y plazas disponibles",
        "step3":              "Te avisamos por Telegram",
        "demo_title":         "Disponibilidad en vivo",
        "refuges_title":      "Refugios cubiertos",
        "sample_free_spots":  "Plazas libres",
        "sample_in":          "en",
        "try":                "Probar",
        "subscribe_hint":     "Recomendado: suscríbete por Telegram en un clic — pulsa el botón de arriba y envía /start. Si ya conoces tu Chat ID, completa el formulario abajo.",
        "chat_id_hint":       "¿No sabes tu Chat ID?",
        "chat_id_how":        "Abre el bot y envía /id",
	},
	"it": {
        "title":              "Disponibilità dei rifugi",
        "last_updated":       "Ultimo aggiornamento",
        "places":             "posti",
        "available":          "Disponibile",
        "full":               "Completo",
        "subscribe":          "Iscriviti",
        "chat_id":            "ID chat Telegram",
        "language":           "Lingua",
        "refuge":             "Rifugio",
        "date_from":          "Data inizio",
        "date_to":            "Data fine",
        "submit":             "Invia",
        "success":            "Iscrizione salvata",
        "hero_title":         "Posti liberi nei rifugi del Monte Bianco — in tempo reale",
        "hero_subtitle":      "Basta controlli quotidiani. Ti avvisiamo quando ci sono posti.",
        "cta_check":          "Controlla disponibilità",
        "cta_subscribe":      "Iscriviti agli avvisi",
        "how_it_works_title": "Come funziona",
        "step1":              "Monitoriamo i siti ufficiali 24/7",
        "step2":              "Mostriamo date e posti disponibili",
        "step3":              "Ti avvisiamo su Telegram",
        "demo_title":         "Disponibilità live",
        "refuges_title":      "Rifugi coperti",
        "sample_free_spots":  "Posti liberi",
        "sample_in":          "a",
        "try":                "Provare",
        "subscribe_hint":     "Consigliato: iscriviti via Telegram in un clic — premi il pulsante sopra e invia /start. Se conosci già il tuo Chat ID, compila il form qui sotto.",
        "chat_id_hint":       "Non conosci il tuo Chat ID?",
        "chat_id_how":        "Apri il bot e invia /id",
	},
}

func T(lang, key string) string {
	if m, ok := supported[lang]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	if v, ok := supported["en"][key]; ok {
		return v
	}
	return key
}

func DetectLang(r *http.Request) string {
	// order: query param -> cookie -> header -> default
	if v := r.URL.Query().Get("lang"); v != "" {
		return normalize(v)
	}
	if c, err := r.Cookie("lang"); err == nil && c != nil {
		return normalize(c.Value)
	}
	al := r.Header.Get("Accept-Language")
	if al != "" {
		for _, part := range strings.Split(al, ",") {
			code := strings.TrimSpace(strings.Split(part, ";")[0])
			code = normalize(code)
			if _, ok := supported[code]; ok {
				return code
			}
			if len(code) >= 2 {
				base := code[:2]
				if _, ok := supported[base]; ok {
					return base
				}
			}
		}
	}
	return "en"
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) >= 2 {
		s = s[:2]
	}
	return s
}
