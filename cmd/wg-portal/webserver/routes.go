package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	csrf "github.com/utrack/gin-csrf"
)

func (s *Server) SetupRoutes() {
	csrfMiddleware := csrf.Middleware(csrf.Options{
		Secret: s.config.Core.SessionSecret,
		ErrorFunc: func(c *gin.Context) {
			c.String(400, "CSRF token mismatch")
			c.Abort()
		},
	})

	// Startpage
	s.server.GET("/", s.GetIndex)
	s.server.GET("/favicon.ico", func(c *gin.Context) {
		file, _ := Statics.ReadFile("assets/img/favicon.ico")
		c.Data(
			http.StatusOK,
			"image/x-icon",
			file,
		)
	})

	// Auth routes
	auth := s.server.Group("/auth")
	auth.Use(csrfMiddleware)
	auth.GET("/login", s.GetLogin)
	auth.POST("/login", s.PostLogin)
	auth.GET("/logout", s.GetLogout)

	// Admin routes
	admin := s.server.Group("/admin")
	admin.Use(csrfMiddleware)
	admin.Use(s.RequireAuthentication("admin"))
	admin.GET("/", s.GetAdminIndex)
	admin.GET("/device/edit", s.GetAdminEditInterface)
	admin.POST("/device/edit", s.PostAdminEditInterface)
	admin.GET("/device/download", s.GetInterfaceConfig)
	admin.GET("/device/write", s.GetSaveConfig)
	admin.GET("/device/applyglobals", s.GetApplyGlobalConfig)
	admin.GET("/peer/edit", s.GetAdminEditPeer)
	admin.POST("/peer/edit", s.PostAdminEditPeer)
	admin.GET("/peer/create", s.GetAdminCreatePeer)
	admin.POST("/peer/create", s.PostAdminCreatePeer)
	admin.GET("/peer/createldap", s.GetAdminCreateLdapPeers)
	admin.POST("/peer/createldap", s.PostAdminCreateLdapPeers)
	admin.GET("/peer/delete", s.GetAdminDeletePeer)
	admin.GET("/peer/download", s.GetPeerConfig)
	admin.GET("/peer/email", s.GetPeerConfigMail)
	admin.GET("/peer/emailall", s.GetAdminSendEmails)

	admin.GET("/users/", s.GetAdminUsersIndex)
	admin.GET("/users/create", s.GetAdminUsersCreate)
	admin.POST("/users/create", s.PostAdminUsersCreate)
	admin.GET("/users/edit", s.GetAdminUsersEdit)
	admin.GET("/users/delete", s.GetAdminUsersDelete)
	admin.POST("/users/edit", s.PostAdminUsersEdit)

	// User routes
	user := s.server.Group("/user")
	user.Use(csrfMiddleware)
	user.Use(s.RequireAuthentication("")) // empty scope = all logged in users
	user.GET("/qrcode", s.GetPeerQRCode)
	user.GET("/profile", s.GetUserIndex)
	user.GET("/download", s.GetPeerConfig)
	user.GET("/email", s.GetPeerConfigMail)
	user.GET("/status", s.GetPeerStatus)

	user.GET("/peer/create", s.GetUserCreatePeer)
	user.POST("/peer/create", s.PostUserCreatePeer)
	user.GET("/peer/edit", s.GetUserEditPeer)
	user.POST("/peer/edit", s.PostUserEditPeer)

}

func (s *Server) RequireAuthentication(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		session := GetSessionData(c)

		if !session.LoggedIn {
			// Abort the request with the appropriate error code
			c.Abort()
			c.Redirect(http.StatusSeeOther, "/auth/login?err=loginreq")
			return
		}

		if scope == "admin" && !session.IsAdmin {
			// Abort the request with the appropriate error code
			c.Abort()
			s.GetHandleError(c, http.StatusUnauthorized, "unauthorized", "not enough permissions")
			return
		}

		// default case if some random scope was set...
		if scope != "" && !session.IsAdmin {
			// Abort the request with the appropriate error code
			c.Abort()
			s.GetHandleError(c, http.StatusUnauthorized, "unauthorized", "not enough permissions")
			return
		}

		// Check if logged-in user is still valid
		if !s.isUserStillValid(session.Email) {
			_ = DestroySessionData(c)
			c.Abort()
			s.GetHandleError(c, http.StatusUnauthorized, "unauthorized", "session no longer available")
			return
		}

		// Continue down the chain to handler etc
		c.Next()
	}
}
