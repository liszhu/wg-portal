package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/h44z/wg-portal/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/h44z/wg-portal/internal/core"
)

type authenticationApiHandler struct {
	backend core.WgPortal
	session SessionStore
}

func (h *authenticationApiHandler) GetExternalLoginProviders() gin.HandlerFunc {
	return func(c *gin.Context) {
		providers := h.backend.GetExternalLoginProviders(c.Request.Context())

		c.JSON(http.StatusOK, providers)
	}
}

func (h *authenticationApiHandler) GetSessionInfo() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentSession := h.session.GetData(c)

		var loggedInUid *string
		if currentSession.LoggedIn {
			uid := string(currentSession.UserIdentifier)
			loggedInUid = &uid
		}

		c.JSON(http.StatusOK, SessionInfoResponse{
			LoggedIn: currentSession.LoggedIn,
			IsAdmin:  currentSession.IsAdmin,
			UserId:   loggedInUid,
		})
	}
}

func (h *authenticationApiHandler) GetOauthInitiate() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentSession := h.session.GetData(c)
		if currentSession.LoggedIn {
			c.JSON(http.StatusBadRequest, GenericResponse{Message: "already logged in"})
			return
		}

		provider := c.Param("provider")

		authCodeUrl, state, nonce, err := h.backend.OauthLoginStep1(c.Request.Context(), provider)
		if err != nil {
			c.JSON(http.StatusInternalServerError, GenericResponse{Message: err.Error()})
			return
		}

		authSession := h.session.DefaultSessionData()
		authSession.OauthState = state
		authSession.OauthNonce = nonce
		authSession.OauthProvider = provider
		h.session.SetData(c, authSession)

		c.JSON(http.StatusOK, OauthInitiationResponse{
			RedirectUrl: authCodeUrl,
			State:       state,
		})
	}
}

func (h *authenticationApiHandler) GetOauthCallback() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentSession := h.session.GetData(c)
		if currentSession.LoggedIn {
			c.JSON(http.StatusBadRequest, GenericResponse{Message: "already logged in"})
			return
		}

		provider := c.Param("provider")
		oauthCode := c.Query("code")
		oauthState := c.Query("state")

		if provider != currentSession.OauthProvider {
			c.JSON(http.StatusBadRequest, GenericResponse{Message: "invalid oauth provider"})
			return
		}
		if oauthState != currentSession.OauthState {
			c.JSON(http.StatusBadRequest, GenericResponse{Message: "invalid oauth state"})
			return
		}

		user, err := h.backend.OauthLoginStep2(c.Request.Context(), provider, currentSession.OauthNonce, oauthCode)
		if err != nil {
			c.JSON(http.StatusUnauthorized, GenericResponse{Message: err.Error()})
			return
		}

		h.setAuthenticatedUser(c, user)

		c.JSON(http.StatusOK, user)
	}
}

func (h *authenticationApiHandler) PostLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentSession := h.session.GetData(c)
		if currentSession.LoggedIn {
			c.JSON(http.StatusOK, GenericResponse{Message: "already logged in"})
			return
		}

		username := strings.ToLower(c.PostForm("username"))
		password := c.PostForm("password")

		user, err := h.backend.PlainLogin(c.Request.Context(), username, password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, GenericResponse{Message: "login failed"})
			return
		}

		h.setAuthenticatedUser(c, user)

		c.JSON(http.StatusOK, user)
	}
}

func (h *authenticationApiHandler) GetLogout() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentSession := h.session.GetData(c)

		if !currentSession.LoggedIn { // Not logged in
			c.JSON(http.StatusOK, GenericResponse{Message: "not logged in"})
			return
		}

		h.session.DestroyData(c)
		c.JSON(http.StatusOK, GenericResponse{Message: "logout ok"})
	}
}

func (h *authenticationApiHandler) AuthenticationMiddleware(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := h.session.GetData(c)

		if !session.LoggedIn {
			// Abort the request with the appropriate error code
			c.Abort()
			c.JSON(http.StatusUnauthorized, GenericResponse{"not logged in"})
			return
		}

		if scope == "admin" && !session.IsAdmin {
			// Abort the request with the appropriate error code
			c.Abort()
			c.JSON(http.StatusForbidden, GenericResponse{"not enough permissions"})
			return
		}

		// default case if some random scope was set...
		if scope != "" && !session.IsAdmin {
			// Abort the request with the appropriate error code
			c.Abort()
			c.JSON(http.StatusForbidden, GenericResponse{"not enough permissions"})
			return
		}

		// Check if logged-in user is still valid
		if !h.isUserStillValid(c.Request.Context(), session.UserIdentifier) {
			h.session.DestroyData(c)
			c.Abort()
			c.String(http.StatusUnauthorized, "unauthorized: session no longer available")
			return
		}

		// Continue down the chain to handler etc
		c.Next()
	}
}

func (h *authenticationApiHandler) isUserStillValid(ctx context.Context, id model.UserIdentifier) bool {
	if user, err := h.backend.GetUser(ctx, id); err != nil || user.IsDisabled() {
		return false
	}
	return true
}

func (h *authenticationApiHandler) setAuthenticatedUser(c *gin.Context, user *model.User) {
	currentSession := h.session.GetData(c)

	currentSession.LoggedIn = true
	currentSession.IsAdmin = user.IsAdmin
	currentSession.UserIdentifier = user.Identifier
	currentSession.Firstname = user.Firstname
	currentSession.Lastname = user.Lastname
	currentSession.Email = user.Email

	currentSession.OauthState = ""
	currentSession.OauthNonce = ""
	currentSession.OauthProvider = ""

	h.session.SetData(c, currentSession)
}
