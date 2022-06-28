package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/h44z/wg-portal/internal/core"
)

type frontendApiHandler struct {
	backend core.WgPortal
}

func (h *frontendApiHandler) getPing() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, time.Now())
	}
}
