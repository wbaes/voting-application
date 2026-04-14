package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/wouter/voting-with-draw/internal/i18n"
)

// getLang reads the language from the Gin context (set by LangMiddleware).
func getLang(c *gin.Context) i18n.Language {
	if v, exists := c.Get("lang"); exists {
		if lang, ok := v.(i18n.Language); ok {
			return lang
		}
	}
	return i18n.EN
}

// baseTemplateData returns common template variables needed on every
// voter-facing page: translation function, active language code, available
// languages for the selector, and the current path for the lang switcher.
func baseTemplateData(c *gin.Context) gin.H {
	lang := getLang(c)
	return gin.H{
		"T":           i18n.T(lang),
		"Lang":        string(lang),
		"Languages":   i18n.All,
		"CurrentPath": c.Request.URL.Path,
	}
}
