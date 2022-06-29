package main

import (
	"bytes"
	"net/http"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/h44z/wg-portal/internal/core"
)

type frontendApiHandler struct {
	cfg     *Config
	backend core.WgPortal

	tpl *template.Template
}

func newFrontendApiHandler(cfg *Config, backend core.WgPortal) *frontendApiHandler {
	h := &frontendApiHandler{cfg: cfg, backend: backend}
	h.parseTemplates()

	return h
}

func (h *frontendApiHandler) parseTemplates() {
	h.tpl = template.Must(template.ParseFS(frontendJs, "frontend_config.js.gotpl"))
}

func (h *frontendApiHandler) GetPing() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, time.Now())
	}
}

func (h *frontendApiHandler) GetFrontendConfigJs() gin.HandlerFunc {
	return func(c *gin.Context) {
		buf := &bytes.Buffer{}
		err := h.tpl.ExecuteTemplate(buf, "frontend_config.js.gotpl", gin.H{
			"BackendUrl": h.cfg.Backend.Core.ExternalUrl,
		})
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		c.Data(http.StatusOK, "application/javascript", buf.Bytes())
	}
}
