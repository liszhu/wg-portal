package main

import (
	"context"
	"syscall"

	"github.com/h44z/wg-portal/internal/core"

	"github.com/h44z/wg-portal/internal"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := internal.SignalAwareContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	logrus.Infof("Starting WireGuard Portal server, version %s...", internal.Version)

	cfg, err := core.LoadConfig()
	internal.AssertNoError(err)

	portal, err := core.NewWgPortal(cfg)
	internal.AssertNoError(err)

	logrus.Info("Started WireGuard Portal server")

	go portal.RunBackgroundTasks(ctx)

	/*fmt.Println("All Users:")
	users, err := portal.GetUsers(ctx, nil)
	internal.AssertNoError(err)
	for i, user := range users {
		fmt.Println(i, user)
	}

	fmt.Println("Paged Users 1:")
	usersPaged, err := portal.GetUsers(ctx, core.UserSearchOptions().WithPageSize(2))
	internal.AssertNoError(err)
	for i, user := range usersPaged {
		fmt.Println(i, user)
	}

	fmt.Println("Paged Users 2:")
	usersPaged2, err := portal.GetUsers(ctx, core.UserSearchOptions().WithPageSize(2).WithPageOffset(2))
	internal.AssertNoError(err)
	for i, user := range usersPaged2 {
		fmt.Println(i, user)
	}*/

	// wait until context gets cancelled
	<-ctx.Done()

	logrus.Info("Stopped WireGuard Portal server")
}
