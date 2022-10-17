package main

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/h44z/wg-portal/internal/app"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
	"github.com/h44z/wg-portal/internal"
	"github.com/sirupsen/logrus"
	ginlogrus "github.com/toorop/gin-logrus"
)

type webServer struct {
	cfg      *Config
	server   *gin.Engine
	hostname string

	*authenticationApiHandler
	*restApiHandler
	*frontendApiHandler
}

func NewServer(cfg *Config, backend *app.App) (*webServer, error) {
	sessionStore := GinSessionStore{sessionIdentifier: "wgPortalSession"}
	s := &webServer{
		cfg:                      cfg,
		authenticationApiHandler: &authenticationApiHandler{backend: backend, session: sessionStore},
		restApiHandler:           &restApiHandler{backend: backend},
		frontendApiHandler:       newFrontendApiHandler(cfg, backend),
	}

	s.setupLogging()
	s.setupHostname()
	s.setupGin()
	s.setupAuthenticationApiRoutes()
	s.setupFrontendApiRoutes()
	s.setupRestApiRoutes()

	return s, nil
}

func (s *webServer) setupLogging() {
	switch s.cfg.Backend.Advanced.LogLevel {
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info", "normal":
		logrus.SetLevel(logrus.InfoLevel)
	case "warning", "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error", "fatal", "critical":
		logrus.SetLevel(logrus.ErrorLevel)
	}
}

func (s *webServer) setupHostname() {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "apiserver"
	}
	s.hostname = fmt.Sprintf("%s, version %s", hostname, internal.Version)
}

func (s *webServer) setupGin() {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = ioutil.Discard
	s.server = gin.New()
	if s.cfg.Frontend.GinDebug {
		gin.SetMode(gin.DebugMode)
		s.server.Use(ginlogrus.Logger(logrus.StandardLogger()))
	}
	s.server.Use(gin.Recovery()).Use(func(c *gin.Context) {
		c.Writer.Header().Set("X-Served-By", s.hostname)
		c.Next()
	})
	cookieStore := memstore.NewStore([]byte(s.cfg.Frontend.SessionSecret))
	cookieStore.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400, // auth session is valid for 1 day
		Secure:   strings.HasPrefix(s.cfg.Backend.Web.ExternalUrl, "https"),
		HttpOnly: true,
	})
	s.server.Use(sessions.Sessions("authsession", cookieStore))

	// Serve static files
	s.server.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/app")
	})
	s.server.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/app/favicon.ico")
	})
	s.server.StaticFS("/app", http.FS(fsMust(fs.Sub(frontendStatics, "frontend-dist"))))
}

func (s *webServer) setupAuthenticationApiRoutes() {
	apiGroup := s.server.Group("/auth", s.corsMiddleware())

	apiGroup.GET("/providers", s.authenticationApiHandler.GetExternalLoginProviders())
	apiGroup.GET("/session", s.authenticationApiHandler.GetSessionInfo())
	apiGroup.GET("/login/:provider/init", s.authenticationApiHandler.GetOauthInitiate())
	apiGroup.GET("/login/:provider/callback", s.authenticationApiHandler.GetOauthCallback())

	apiGroup.POST("/login", s.authenticationApiHandler.PostLogin())
	apiGroup.GET("/logout", s.authenticationApiHandler.GetLogout())
}

func (s *webServer) setupRestApiRoutes() {
	apiGroup := s.server.Group("/rest/api/v1", s.corsMiddleware())

	apiGroup.GET("/ping", s.restApiHandler.getPing())
}

func (s *webServer) setupFrontendApiRoutes() {
	publicApiGroup := s.server.Group("/frontend/public", s.corsMiddleware())
	publicApiGroup.GET("/config.js", s.frontendApiHandler.GetFrontendConfigJs())

	apiGroup := s.server.Group("/frontend/api/v1",
		s.corsMiddleware(), s.authenticationApiHandler.AuthenticationMiddleware(""))

	apiGroup.GET("/ping", s.frontendApiHandler.GetPing())

	// Interface routes
	apiGroup.GET("/interfaces", s.frontendApiHandler.GetInterfaces())
	apiGroup.GET("/interfaces/prepare", s.frontendApiHandler.GetFreshInterface())

	// Peer routes
	apiGroup.GET("/peers", s.frontendApiHandler.GetPeers())
	apiGroup.GET("/peers/prepare", s.frontendApiHandler.GetFreshPeer())
}

func (s *webServer) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
	}
}

func (s *webServer) Run(ctx context.Context) {
	logrus.Infof("starting web service on %s", s.cfg.Frontend.ListeningAddress)

	// Run web service
	srv := &http.Server{
		Addr:    s.cfg.Frontend.ListeningAddress,
		Handler: s.server,
	}

	srvContext, cancelFn := context.WithCancel(ctx)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logrus.Infof("web service on %s exited: %v", s.cfg.Frontend.ListeningAddress, err)
			cancelFn()
		}
	}()

	// Wait for the main context to end
	<-srvContext.Done()

	logrus.Debug("web service shutting down, grace period: 5 seconds...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)

	logrus.Debug("web service shut down")
}

// -- helpers

func fsMust(f fs.FS, err error) fs.FS {
	if err != nil {
		panic(err)
	}
	return f
}
