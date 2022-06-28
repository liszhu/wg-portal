package server

import (
	"context"
	"encoding/gob"
	"html/template"
	"io/fs"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
	"github.com/h44z/wg-portal/internal"
	"github.com/h44z/wg-portal/internal/core"
	"github.com/h44z/wg-portal/internal/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	ginlogrus "github.com/toorop/gin-logrus"
)

const SessionIdentifier = "wgPortalSession"

func init() {
	gob.Register(SessionData{})
	gob.Register(FlashData{})
	gob.Register(model.Peer{})
	gob.Register(model.Interface{})
	gob.Register(model.User{})
}

type SessionData struct {
	LoggedIn   bool
	IsAdmin    bool
	Firstname  string
	Lastname   string
	Email      string
	DeviceName model.InterfaceIdentifier

	SortedBy      map[string]string
	SortDirection map[string]string
	Search        map[string]string

	AlertData string
	AlertType string
	FormData  interface{}
}

type FlashData struct {
	HasAlert bool
	Message  string
	Type     string
}

type StaticData struct {
	WebsiteTitle string
	WebsiteLogo  string
	CompanyName  string
	Year         int
	Version      string
}

type Server struct {
	config *core.Config
	server *gin.Engine

	backend core.WgPortal
}

func NewServer(cfg *core.Config, backend core.WgPortal) (*Server, error) {
	s := &Server{config: cfg, backend: backend}

	dir := s.getExecutableDirectory()
	rDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	logrus.Infof("real working directory: %s", rDir)
	logrus.Infof("current working directory: %s", dir)

	// Init rand
	rand.Seed(time.Now().UnixNano())

	// Setup http server
	gin.SetMode(gin.DebugMode)
	gin.DefaultWriter = ioutil.Discard
	s.server = gin.New()
	if logrus.GetLevel() == logrus.TraceLevel {
		s.server.Use(ginlogrus.Logger(logrus.StandardLogger()))
	}
	s.server.Use(gin.Recovery())

	// Authentication cookies
	cookieStore := memstore.NewStore([]byte(s.config.Core.SessionSecret))
	cookieStore.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400, // auth session is valid for 1 day
		Secure:   strings.HasPrefix(s.config.Core.ExternalUrl, "https"),
		HttpOnly: true,
	})
	s.server.Use(sessions.Sessions("authsession", cookieStore))
	s.server.SetFuncMap(template.FuncMap{
		"formatBytes": internal.ByteCountSI,
		"urlEncode":   url.QueryEscape,
		"startsWith":  strings.HasPrefix,
		"userForEmail": func(users []model.User, email string) *model.User {
			for i := range users {
				if users[i].Email == email {
					return &users[i]
				}
			}
			return nil
		},
	})

	// Setup templates
	templates := template.Must(template.New("").Funcs(s.server.FuncMap).ParseFS(Templates, "assets/tpl/*.html"))
	s.server.SetHTMLTemplate(templates)

	// Serve static files
	s.server.StaticFS("/css", http.FS(fsMust(fs.Sub(Statics, "assets/css"))))
	s.server.StaticFS("/js", http.FS(fsMust(fs.Sub(Statics, "assets/js"))))
	s.server.StaticFS("/img", http.FS(fsMust(fs.Sub(Statics, "assets/img"))))
	s.server.StaticFS("/fonts", http.FS(fsMust(fs.Sub(Statics, "assets/fonts"))))

	// Setup all routes
	s.SetupRoutes()

	logrus.Infof("setup of service completed!")

	return s, nil
}

func (s *Server) Run(ctx context.Context) {
	logrus.Infof("starting web service on %s", s.config.Core.ListeningAddress)

	// Run web service
	srv := &http.Server{
		Addr:    s.config.Core.ListeningAddress,
		Handler: s.server,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logrus.Debugf("web service on %s exited: %v", s.config.Core.ListeningAddress, err)
		}
	}()

	<-ctx.Done()

	logrus.Debug("web service shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)

	logrus.Debug("web service shut down")
}

func (s *Server) getExecutableDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		logrus.Errorf("failed to get executable directory: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "assets")); os.IsNotExist(err) {
		return "." // assets directory not found -> we are developing in goland =)
	}

	return dir
}

func (s *Server) getStaticData() StaticData {
	return StaticData{
		WebsiteTitle: s.config.Core.Title,
		WebsiteLogo:  s.config.Core.LogoUrl,
		CompanyName:  s.config.Core.CompanyName,
		Year:         time.Now().Year(),
		Version:      internal.Version,
	}
}

func GetSessionData(c *gin.Context) SessionData {
	session := sessions.Default(c)
	rawSessionData := session.Get(SessionIdentifier)

	var sessionData SessionData
	if rawSessionData != nil {
		sessionData = rawSessionData.(SessionData)
	} else {
		sessionData = SessionData{
			Search:        map[string]string{"peers": "", "userpeers": "", "users": ""},
			SortedBy:      map[string]string{"peers": "handshake", "userpeers": "id", "users": "email"},
			SortDirection: map[string]string{"peers": "desc", "userpeers": "asc", "users": "asc"},
			Email:         "",
			Firstname:     "",
			Lastname:      "",
			DeviceName:    "",
			IsAdmin:       false,
			LoggedIn:      false,
		}
		session.Set(SessionIdentifier, sessionData)
		if err := session.Save(); err != nil {
			logrus.Errorf("failed to store session: %v", err)
		}
	}

	return sessionData
}

func GetFlashes(c *gin.Context) []FlashData {
	session := sessions.Default(c)
	flashes := session.Flashes()
	if err := session.Save(); err != nil {
		logrus.Errorf("failed to store session after setting flash: %v", err)
	}

	flashData := make([]FlashData, len(flashes))
	for i := range flashes {
		flashData[i] = flashes[i].(FlashData)
	}

	return flashData
}

func UpdateSessionData(c *gin.Context, data SessionData) error {
	session := sessions.Default(c)
	session.Set(SessionIdentifier, data)
	if err := session.Save(); err != nil {
		logrus.Errorf("failed to store session: %v", err)
		return errors.Wrap(err, "failed to store session")
	}
	return nil
}

func DestroySessionData(c *gin.Context) error {
	session := sessions.Default(c)
	session.Delete(SessionIdentifier)
	if err := session.Save(); err != nil {
		logrus.Errorf("failed to destroy session: %v", err)
		return errors.Wrap(err, "failed to destroy session")
	}
	return nil
}

func SetFlashMessage(c *gin.Context, message, typ string) {
	session := sessions.Default(c)
	session.AddFlash(FlashData{
		Message: message,
		Type:    typ,
	})
	if err := session.Save(); err != nil {
		logrus.Errorf("failed to store session after setting flash: %v", err)
	}
}

func (s SessionData) GetSortIcon(table, field string) string {
	if s.SortedBy[table] != field {
		return "fa-sort"
	}
	if s.SortDirection[table] == "asc" {
		return "fa-sort-alpha-down"
	} else {
		return "fa-sort-alpha-up"
	}
}

func fsMust(f fs.FS, err error) fs.FS {
	if err != nil {
		panic(err)
	}
	return f
}
