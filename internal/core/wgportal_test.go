package core

import (
	"context"
	"testing"

	"github.com/h44z/wg-portal/internal/model"
	userMock "github.com/h44z/wg-portal/internal/user/mocks"
	"github.com/stretchr/testify/assert"
)

func Test_wgPortal_GetUsers_All(t *testing.T) {
	userManager := &userMock.Manager{}

	portal := wgPortal{
		users: userManager,
	}

	userManager.On("GetAllUsers").Return([]*model.User{{Identifier: "uid1"}, {Identifier: "uid2"}}, nil)

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

	userManager.On("GetAllUsers").Return([]*model.User{{Identifier: "3"}, {Identifier: "1"}, {Identifier: "2"}}, nil)

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

	userManager.On("GetAllUsers").Return([]*model.User{{Identifier: "3"}, {Identifier: "1"}, {Identifier: "2"}}, nil)

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

	userManager.On("GetAllUsers").Return([]*model.User{{Identifier: "3"}, {Identifier: "1"}, {Identifier: "2"}}, nil)
	users, err := portal.GetUsers(context.Background(), UserSearchOptions().WithPageSize(2))
	assert.NoError(t, err)
	assert.Equal(t, []model.User{{Identifier: "1"}, {Identifier: "2"}}, users)

	userManager.On("GetAllUsers").Return([]*model.User{{Identifier: "3"}, {Identifier: "1"}, {Identifier: "2"}}, nil)
	users, err = portal.GetUsers(context.Background(), UserSearchOptions().WithPageSize(2).WithPageOffset(2))
	assert.NoError(t, err)
	assert.Equal(t, []model.User{{Identifier: "3"}}, users)

	userManager.On("GetAllUsers").Return([]*model.User{{Identifier: "3"}, {Identifier: "1"}, {Identifier: "2"}}, nil)
	users, err = portal.GetUsers(context.Background(), UserSearchOptions().WithPageSize(2).WithPageOffset(3))
	assert.Error(t, err)

	userManager.AssertExpectations(t)
}
