package main

import (
	"context"
	"io/fs"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	ginlogrus "github.com/toorop/gin-logrus"
)

type webServer struct {
	server *gin.Engine
}

func NewServer(authMiddleware gin.HandlerFunc) (*webServer, error) {
	s := &webServer{}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "apiserver"
	}
	hostname += ", version 1.0"

	// Setup http server
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = ioutil.Discard
	s.server = gin.New()
	if logrus.GetLevel() == logrus.TraceLevel {
		gin.SetMode(gin.DebugMode)
		s.server.Use(ginlogrus.Logger(logrus.StandardLogger()))
	}
	s.server.Use(gin.Recovery()).Use(func(c *gin.Context) {
		c.Writer.Header().Set("X-Served-By", hostname)
		c.Next()
	})
	s.server.Use(authMiddleware)

	// Serve static files
	s.server.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/app")
	})
	s.server.StaticFS("/app", http.FS(fsMust(fs.Sub(frontendStatics, "frontend-dist"))))

	// Fix for Windows systems (See: https://github.com/golang/go/issues/32350)
	if runtime.GOOS == "windows" {
		if err := mime.AddExtensionType(".js", "application/javascript"); err != nil {
			log.Fatalf("could not add mime extension type: %v", err)
		}
	}

	return s, nil

}

func (s *webServer) Run(ctx context.Context, listenAddress string) {
	logrus.Infof("starting web service on %s", listenAddress)

	// Run web service
	srv := &http.Server{
		Addr:    listenAddress,
		Handler: s.server,
	}

	srvContext, cancelFn := context.WithCancel(ctx)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logrus.Infof("web service on %s exited: %v", listenAddress, err)
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
