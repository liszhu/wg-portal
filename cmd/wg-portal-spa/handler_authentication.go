package main

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/h44z/wg-portal/internal/app"

	"github.com/gin-gonic/gin"
	"github.com/h44z/wg-portal/internal/model"
)

type authenticationApiHandler struct {
	backend *app.App
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
		var firstname *string
		var lastname *string
		var email *string
		if currentSession.LoggedIn {
			uid := string(currentSession.UserIdentifier)
			f := currentSession.Firstname
			l := currentSession.Lastname
			e := currentSession.Email
			loggedInUid = &uid
			firstname = &f
			lastname = &l
			email = &e
		}

		c.JSON(http.StatusOK, SessionInfoResponse{
			LoggedIn:       currentSession.LoggedIn,
			IsAdmin:        currentSession.IsAdmin,
			UserIdentifier: loggedInUid,
			UserFirstname:  firstname,
			UserLastname:   lastname,
			UserEmail:      email,
		})
	}
}

func (h *authenticationApiHandler) GetOauthInitiate() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentSession := h.session.GetData(c)

		autoRedirect, _ := strconv.ParseBool(c.DefaultQuery("redirect", "false"))
		returnTo := c.Query("return")
		provider := c.Param("provider")

		var returnUrl *url.URL
		var returnParams string
		redirectToReturn := func() {
			c.Redirect(http.StatusFound, returnUrl.String()+"?"+returnParams)
		}

		if returnTo != "" {
			if u, err := url.Parse(returnTo); err == nil {
				returnUrl = u
			}
			queryParams := returnUrl.Query()
			queryParams.Set("wgLoginState", "err") // by default, we set the state to error
			returnUrl.RawQuery = ""                // remove potential query params
			returnParams = queryParams.Encode()
		}

		if currentSession.LoggedIn {
			if autoRedirect {
				queryParams := returnUrl.Query()
				queryParams.Set("wgLoginState", "success")
				returnParams = queryParams.Encode()
				redirectToReturn()
			} else {
				c.JSON(http.StatusBadRequest, GenericResponse{Message: "already logged in"})
			}
			return
		}

		authCodeUrl, state, nonce, err := h.backend.OauthLoginStep1(c.Request.Context(), provider)
		if err != nil {
			if autoRedirect {
				redirectToReturn()
			} else {
				c.JSON(http.StatusInternalServerError, GenericResponse{Message: err.Error()})
			}
			return
		}

		authSession := h.session.DefaultSessionData()
		authSession.OauthState = state
		authSession.OauthNonce = nonce
		authSession.OauthProvider = provider
		authSession.OauthReturnTo = returnTo
		h.session.SetData(c, authSession)

		if autoRedirect {
			c.Redirect(http.StatusFound, authCodeUrl)
		} else {
			c.JSON(http.StatusOK, OauthInitiationResponse{
				RedirectUrl: authCodeUrl,
				State:       state,
			})
		}
	}
}

func (h *authenticationApiHandler) GetOauthCallback() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentSession := h.session.GetData(c)

		var returnUrl *url.URL
		var returnParams string
		redirectToReturn := func() {
			c.Redirect(http.StatusFound, returnUrl.String()+"?"+returnParams)
		}

		if currentSession.OauthReturnTo != "" {
			if u, err := url.Parse(currentSession.OauthReturnTo); err == nil {
				returnUrl = u
			}
			queryParams := returnUrl.Query()
			queryParams.Set("wgLoginState", "err") // by default, we set the state to error
			returnUrl.RawQuery = ""                // remove potential query params
			returnParams = queryParams.Encode()
		}

		if currentSession.LoggedIn {
			if returnUrl != nil {
				queryParams := returnUrl.Query()
				queryParams.Set("wgLoginState", "success")
				returnParams = queryParams.Encode()
				redirectToReturn()
			} else {
				c.JSON(http.StatusBadRequest, GenericResponse{Message: "already logged in"})
			}
			return
		}

		provider := c.Param("provider")
		oauthCode := c.Query("code")
		oauthState := c.Query("state")

		if provider != currentSession.OauthProvider {
			if returnUrl != nil {
				redirectToReturn()
			} else {
				c.JSON(http.StatusBadRequest, GenericResponse{Message: "invalid oauth provider"})
			}
			return
		}
		if oauthState != currentSession.OauthState {
			if returnUrl != nil {
				redirectToReturn()
			} else {
				c.JSON(http.StatusBadRequest, GenericResponse{Message: "invalid oauth state"})
			}
			return
		}

		user, err := h.backend.OauthLoginStep2(c.Request.Context(), provider, currentSession.OauthNonce, oauthCode)
		if err != nil {
			if returnUrl != nil {
				redirectToReturn()
			} else {
				c.JSON(http.StatusUnauthorized, GenericResponse{Message: err.Error()})
			}
			return
		}

		h.setAuthenticatedUser(c, user)

		if returnUrl != nil {
			queryParams := returnUrl.Query()
			queryParams.Set("wgLoginState", "success")
			returnParams = queryParams.Encode()
			redirectToReturn()
		} else {
			c.JSON(http.StatusOK, user)
		}
	}
}

func (h *authenticationApiHandler) PostLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentSession := h.session.GetData(c)
		if currentSession.LoggedIn {
			c.JSON(http.StatusOK, GenericResponse{Message: "already logged in"})
			return
		}

		var loginData struct {
			Username string `json:"username" binding:"required,min=2"`
			Password string `json:"password" binding:"required,min=4"`
		}

		if err := c.ShouldBindJSON(&loginData); err != nil {
			c.JSON(http.StatusBadRequest, GenericResponse{Message: err.Error()})
			return
		}

		user, err := h.backend.PlainLogin(c.Request.Context(), loginData.Username, loginData.Password)
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
	/*if user, err := h.backend.GetUser(ctx, id); err != nil || user.IsDisabled() {
		return false
	}*/
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
	currentSession.OauthReturnTo = ""

	h.session.SetData(c, currentSession)
}
