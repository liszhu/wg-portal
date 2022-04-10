package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/h44z/wg-portal/internal/adapters/auth"
	"github.com/h44z/wg-portal/internal/config"
	"github.com/h44z/wg-portal/internal/domain"
	"github.com/h44z/wg-portal/internal/ports"
	"github.com/h44z/wg-portal/internal/service"
	"github.com/urfave/cli/v2"
)

const (
	dsnFlag       = "dsn"
	interfaceFlag = "interface"
)

type cliApp struct {
	app *cli.App

	authenticator ports.Authenticator
}

func NewCliApp(authenticator ports.Authenticator) *cliApp {
	a := &cliApp{
		app:           cli.NewApp(),
		authenticator: authenticator,
	}

	a.app.Name = "wg-portal"
	a.app.Version = "0.0.1"
	a.app.Usage = "WireGuard Portal CLI client"
	a.app.EnableBashCompletion = true
	a.app.Commands = a.Commands()
	a.app.Flags = a.Flags()
	a.app.Before = func(c *cli.Context) error {
		dsn := c.String(dsnFlag)

		fmt.Println("DSN:", dsn)
		return nil
	}
	return a
}

func (a *cliApp) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  dsnFlag,
			Value: "./sqlite.db",
			Usage: "A DSN for the data store.",
		},
	}
}

func (a *cliApp) Commands() []*cli.Command {
	return []*cli.Command{
		{
			Name:      "plainauth",
			Aliases:   []string{"p"},
			Usage:     "authenticate",
			ArgsUsage: "<authenticator identifier> <username> <password>",
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 3 {
					return errors.New("missing/invalid parameters, usage: <authenticator identifier> <username> <password>")
				}
				authenticatorIdentifier := domain.AuthenticatorId(strings.TrimSpace(c.Args().Get(0)))
				username := strings.TrimSpace(c.Args().Get(1))
				password := strings.TrimSpace(c.Args().Get(2))

				cfg, err := a.authenticator.GetAuthenticator(authenticatorIdentifier)
				if err != nil {
					return err
				}

				if cfg.GetType() != domain.AuthenticatorTypePlain {
					return errors.New("wrong authenticator type")
				}

				fmt.Println("Testing authenticator", cfg.GetName(), "of type", cfg.GetType())

				ctx := context.Background()
				fmt.Println("Authenticated:", a.authenticator.IsAuthenticated(ctx))
				ctx, err = a.authenticator.AuthenticateContext(context.Background(), authenticatorIdentifier,
					username, password)
				if err != nil {
					return err
				}
				fmt.Println("Authenticated:", a.authenticator.IsAuthenticated(ctx))
				fmt.Println(a.authenticator.GetUserInfo(ctx))

				return nil
			},
		},
		{
			Name:      "oauth",
			Aliases:   []string{"p"},
			Usage:     "authenticate against oauth",
			ArgsUsage: "<authenticator identifier>",
			Action: func(c *cli.Context) error {
				if c.Args().Len() != 1 {
					return errors.New("missing/invalid parameters, usage: <authenticator identifier>")
				}
				authenticatorIdentifier := domain.AuthenticatorId(strings.TrimSpace(c.Args().Get(0)))

				cfg, err := a.authenticator.GetAuthenticator(authenticatorIdentifier)
				if err != nil {
					return err
				}
				if cfg.GetType() != domain.AuthenticatorTypeOAuth && cfg.GetType() != domain.AuthenticatorTypeOidc {
					return errors.New("wrong authenticator type")
				}

				fmt.Println("Testing authenticator", cfg.GetName(), "of type", cfg.GetType())

				ctx := context.Background()
				fmt.Println("Authenticated:", a.authenticator.IsAuthenticated(ctx))
				url, state, nonce, err := a.authenticator.GetOauthUrl(authenticatorIdentifier)
				if err != nil {
					return err
				}

				fmt.Println("Generated state:", state, "and nonce:", nonce)
				fmt.Println("Please visit the following URL:", url)
				fmt.Println("Finish the authentication steps, you will recieve a code, state and nonce parameter.")
				fmt.Println("Ensure that the retrieved state and nonce is the same as stated above!")
				fmt.Println("Copy and paste the retrieved code here:")

				var code string
				_, err = fmt.Scanln(&code)
				if err != nil {
					return err
				}

				ctx, err = a.authenticator.AuthenticateContextWithCode(ctx, authenticatorIdentifier, code, state, nonce)
				if err != nil {
					return err
				}
				fmt.Println("Authenticated:", a.authenticator.IsAuthenticated(ctx))
				fmt.Println(a.authenticator.GetUserInfo(ctx))

				return nil
			},
		},
	}
}

func (a *cliApp) Run() error {
	return a.app.Run(os.Args)
}

func main() {
	cfg, err := config.Load()
	assertNoError(err)

	var plainAuth []ports.PlainAuthenticatorRepository
	var oauthAuth []ports.OauthAuthenticatorRepository

	for i := range cfg.Auth.Ldap {
		authenticator, err := auth.NewLdapAuthenticator(&cfg.Auth.Ldap[i])
		assertNoError(err)
		plainAuth = append(plainAuth, authenticator)
	}
	for i := range cfg.Auth.OAuth {
		authenticator, err := auth.NewOauthAuthenticator("http://localhost:8080/auth/", &cfg.Auth.OAuth[i])
		assertNoError(err)
		oauthAuth = append(oauthAuth, authenticator)
	}
	for i := range cfg.Auth.OpenIDConnect {
		authenticator, err := auth.NewOidcAuthenticator("http://localhost:8080/auth/", &cfg.Auth.OpenIDConnect[i])
		assertNoError(err)
		oauthAuth = append(oauthAuth, authenticator)
	}

	authSvc, err := service.NewAuthenticatorService(plainAuth, oauthAuth)
	assertNoError(err)

	fmt.Println("Registered", len(authSvc.GetAuthenticators()), "authentication backends")

	app := NewCliApp(authSvc)

	err = app.Run()
	assertNoError(err)
}

func assertNoError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
