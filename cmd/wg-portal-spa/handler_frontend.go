package main

import (
	"bytes"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/h44z/wg-portal/internal/model"

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

func (h *frontendApiHandler) GetPing() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, time.Now())
	}
}

func (h *frontendApiHandler) GetInterfaces() gin.HandlerFunc {
	return func(c *gin.Context) {
		searchOptions := core.InterfaceSearchOptions()

		interfaces, err := h.backend.GetInterfaces(c.Request.Context(), searchOptions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GenericResponse{Message: err.Error()})
			return
		}

		allInterfaces, err := interfaces.
			Sort(func(a, b *model.Interface) bool { return a.Identifier < b.Identifier }).
			Paginate(0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GenericResponse{Message: err.Error()})
			return
		}

		c.JSON(http.StatusOK, allInterfaces)
	}
}

func (h *frontendApiHandler) GetPeers() gin.HandlerFunc {
	return func(c *gin.Context) {
		interfaceId := c.Query("interface")

		searchOptions := core.PeerSearchOptions().WithInterface(model.InterfaceIdentifier(interfaceId))

		peers, err := h.backend.GetPeers(c.Request.Context(), searchOptions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GenericResponse{Message: err.Error()})
			return
		}

		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pagesize", "25"))

		allPeers, err := peers.
			Sort(func(a, b *model.Peer) bool { return a.Identifier < b.Identifier }).
			Size(pageSize).
			Paginate(offset)
		if err != nil && err != core.ErrNoMorePage {
			c.JSON(http.StatusInternalServerError, GenericResponse{Message: err.Error()})
			return
		}

		max := pageSize
		if pageSize == core.PageSizeAll {
			max = peers.TotalLength()
		}
		finished := len(allPeers) < max // end was reached if we receive a page that is not full
		if err == core.ErrNoMorePage {
			finished = true
		}

		c.JSON(http.StatusOK, PagedResponse[*model.Peer]{Records: allPeers, MoreRecords: !finished})
	}
}

func (h *frontendApiHandler) GetFreshInterface() gin.HandlerFunc {
	return func(c *gin.Context) {
		preparedInterface, err := h.backend.PrepareNewInterface(c.Request.Context(), "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, GenericResponse{Message: err.Error()})
			return
		}

		c.JSON(http.StatusOK, preparedInterface)
	}
}

func (h *frontendApiHandler) GetFreshPeer() gin.HandlerFunc {
	return func(c *gin.Context) {
		interfaceId := c.Query("interface")
		preparedPeer, err := h.backend.PrepareNewPeer(c.Request.Context(), model.InterfaceIdentifier(interfaceId))
		if err != nil {
			c.JSON(http.StatusInternalServerError, GenericResponse{Message: err.Error()})
			return
		}

		c.JSON(http.StatusOK, preparedPeer)
	}
}
