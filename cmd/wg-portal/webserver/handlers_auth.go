package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/h44z/wg-portal/internal/core"
	csrf "github.com/utrack/gin-csrf"
)

func (s *Server) GetLogin(c *gin.Context) {
	currentSession := GetSessionData(c)
	if currentSession.LoggedIn {
		c.Redirect(http.StatusSeeOther, "/") // already logged in
	}

	authError := c.DefaultQuery("err", "")
	errMsg := "Unknown error occurred, try again!"
	switch authError {
	case "missingdata":
		errMsg = "Invalid login data retrieved, please fill out all fields and try again!"
	case "authfail":
		errMsg = "Authentication failed!"
	case "loginreq":
		errMsg = "Login required!"
	}

	c.HTML(http.StatusOK, "login.html", gin.H{
		"error":   authError != "",
		"message": errMsg,
		"static":  s.getStaticData(),
		"Csrf":    csrf.GetToken(c),
	})
}

func (s *Server) PostLogin(c *gin.Context) {
	currentSession := GetSessionData(c)
	if currentSession.LoggedIn {
		// already logged in
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	username := strings.ToLower(c.PostForm("username"))
	password := c.PostForm("password")

	user, err := s.backend.PlainLogin(c.Request.Context(), username, password)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/auth/login?err=authfail")
		return
	}

	defaultInterfaceName := ""
	paginator, err := s.backend.GetInterfaces(c.Request.Context(), core.InterfaceSearchOptions())
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/auth/login?err=nointerfaces")
		return
	}
	interfaces, err := paginator.Paginate(0)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/auth/login?err=nointerfaces")
		return
	}
	if len(interfaces) > 0 {
		defaultInterfaceName = string(interfaces[0].Identifier)
	}

	// Set authenticated session
	sessionData := GetSessionData(c)
	sessionData.LoggedIn = true
	sessionData.IsAdmin = user.IsAdmin
	sessionData.Email = user.Email
	sessionData.Firstname = user.Firstname
	sessionData.Lastname = user.Lastname
	sessionData.DeviceName = defaultInterfaceName

	if err := UpdateSessionData(c, sessionData); err != nil {
		s.GetHandleError(c, http.StatusInternalServerError, "login error", "failed to save session")
		return
	}
	c.Redirect(http.StatusSeeOther, "/")
}

func (s *Server) GetLogout(c *gin.Context) {
	currentSession := GetSessionData(c)

	if !currentSession.LoggedIn { // Not logged in
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	if err := DestroySessionData(c); err != nil {
		s.GetHandleError(c, http.StatusInternalServerError, "logout error", "failed to destroy session")
		return
	}
	c.Redirect(http.StatusSeeOther, "/")
}
