package main

import (
	"net/http"
	"time"

	"github.com/h44z/wg-portal/internal/core"

	"github.com/gin-gonic/gin"
)

type restApiHandler struct {
	backend core.WgPortal
}

func (h *restApiHandler) getPing() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, time.Now())
	}
}
