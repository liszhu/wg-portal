package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/h44z/wg-portal/internal/model"
	userMock "github.com/h44z/wg-portal/internal/user/mocks"
	wgMock "github.com/h44z/wg-portal/internal/wireguard/mocks"
	"github.com/stretchr/testify/assert"
)

func Test_wgPortal_GetUsers_All(t *testing.T) {
	userManager := &userMock.Manager{}

	portal := wgPortal{
		users: userManager,
	}

	userManager.On("GetUsers").Return([]*model.User{{Identifier: "uid1"}, {Identifier: "uid2"}}, nil)

	users, err := portal.GetUsers(context.Background(), nil)
	assert.NoError(t, err)
	assert.Len(t, users, 2)

	userManager.AssertExpectations(t)
}

func Test_wgPortal_GetUsers_All_Sort_ASC(t *testing.T) {
	userManager := &userMock.Manager{}

	portal := wgPortal{
		users: userManager,
	}

	userManager.On("GetUsers").Return([]*model.User{{Identifier: "3"}, {Identifier: "1"}, {Identifier: "2"}}, nil)

	users, err := portal.GetUsers(context.Background(), UserSearchOptions().WithSorting("identifier", SortAsc))
	assert.NoError(t, err)
	assert.Equal(t, []model.User{{Identifier: "1"}, {Identifier: "2"}, {Identifier: "3"}}, users)

	userManager.AssertExpectations(t)
}

func Test_wgPortal_GetUsers_All_Sort_DESC(t *testing.T) {
	userManager := &userMock.Manager{}

	portal := wgPortal{
		users: userManager,
	}

	userManager.On("GetUsers").Return([]*model.User{{Identifier: "3"}, {Identifier: "1"}, {Identifier: "2"}}, nil)

	users, err := portal.GetUsers(context.Background(), UserSearchOptions().WithSorting("identifier", SortDesc))
	assert.NoError(t, err)
	assert.Equal(t, []model.User{{Identifier: "3"}, {Identifier: "2"}, {Identifier: "1"}}, users)

	userManager.AssertExpectations(t)
}

func Test_wgPortal_GetUsers_All_Paging(t *testing.T) {
	userManager := &userMock.Manager{}

	portal := wgPortal{
		users: userManager,
	}

	userManager.On("GetUsers").Return([]*model.User{{Identifier: "3"}, {Identifier: "1"}, {Identifier: "2"}}, nil)
	users, err := portal.GetUsers(context.Background(), UserSearchOptions().WithPageSize(2))
	assert.NoError(t, err)
	assert.Equal(t, []model.User{{Identifier: "1"}, {Identifier: "2"}}, users)

	userManager.On("GetUsers").Return([]*model.User{{Identifier: "3"}, {Identifier: "1"}, {Identifier: "2"}}, nil)
	users, err = portal.GetUsers(context.Background(), UserSearchOptions().WithPageSize(2).WithPageOffset(2))
	assert.NoError(t, err)
	assert.Equal(t, []model.User{{Identifier: "3"}}, users)

	userManager.On("GetUsers").Return([]*model.User{{Identifier: "3"}, {Identifier: "1"}, {Identifier: "2"}}, nil)
	users, err = portal.GetUsers(context.Background(), UserSearchOptions().WithPageSize(2).WithPageOffset(3))
	assert.Error(t, err)

	userManager.AssertExpectations(t)
}

func Test_wgPortal_CreateUser_ConfigDefaultInterfaces(t *testing.T) {
	userManager := &userMock.Manager{}
	wgManager := &wgMock.Manager{}

	portal := wgPortal{
		cfg:   &Config{DefaultPeerInterfaces: []model.InterfaceIdentifier{"wg0", "wg1"}},
		users: userManager,
		wg:    wgManager,
	}

	testUser := &model.User{
		Identifier: "uid1",
	}

	// user creation
	userManager.On("CreateUser", testUser).Return(nil)

	// wg0 peer creation
	wgManager.On("GetInterface", model.InterfaceIdentifier("wg0")).Return(&model.Interface{Identifier: "wg0"}, nil)
	wgManager.On("GetFreshKeypair").Return(model.KeyPair{PublicKey: "0123456789"}, nil)
	wgManager.On("GetPreSharedKey").Return(model.PreSharedKey(""), nil)
	wgManager.On("GetFreshIps", model.InterfaceIdentifier("wg0")).Return("", nil)
	wgManager.On("SavePeers", mock.AnythingOfType("*model.Peer")).Return(nil)

	// wg1 peer creation
	wgManager.On("GetInterface", model.InterfaceIdentifier("wg1")).Return(&model.Interface{Identifier: "wg1"}, nil)
	wgManager.On("GetFreshKeypair").Return(model.KeyPair{PublicKey: "9876543210"}, nil)
	wgManager.On("GetPreSharedKey").Return(model.PreSharedKey(""), nil)
	wgManager.On("GetFreshIps", model.InterfaceIdentifier("wg1")).Return("", nil)
	wgManager.On("SavePeers", mock.AnythingOfType("*model.Peer")).Return(nil)

	createdUser, err := portal.CreateUser(context.Background(), testUser, UserCreateOptions().WithDefaultPeer(true))
	assert.NoError(t, err)
	assert.Equal(t, testUser, createdUser)

	userManager.AssertExpectations(t)
	wgManager.AssertExpectations(t)
}
