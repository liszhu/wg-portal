package core

import (
	"context"
	"io"

	"github.com/h44z/wg-portal/internal/model"
)

type WgPortal interface {
	RunBackgroundTasks(context.Context)

	GetUsers(context.Context, *userSearchOptions) ([]model.User, error)
	GetUserIds(context.Context, *userSearchOptions) ([]model.UserIdentifier, error)
	CreateUser(context.Context, *model.User, *userCreateOptions) (*model.User, error)
	UpdateUser(context.Context, *model.User, *userUpdateOptions) (*model.User, error)
	DeleteUser(context.Context, model.UserIdentifier, *userDeleteOptions) error

	GetInterfaces(context.Context, *interfaceSearchOptions) ([]model.Interface, error)
	CreateInterface(context.Context, *model.Interface) (*model.Interface, error)
	UpdateInterface(context.Context, *model.Interface) (*model.Interface, error)
	DeleteInterface(context.Context, model.InterfaceIdentifier) error
	PrepareNewInterface(context.Context, model.InterfaceIdentifier) (*model.Interface, error)
	GetInterfaceWgQuickConfig(context.Context, model.InterfaceIdentifier) (io.Reader, error)
	ApplyGlobalSettings(context.Context, model.InterfaceIdentifier) error

	GetImportableInterfaces(context.Context, *interfaceSearchOptions) ([]model.ImportableInterface, error)
	ImportInterface(context.Context, *model.ImportableInterface, *importOptions) (*model.Interface, error)

	GetPeers(context.Context, *peerSearchOptions) ([]model.Peer, error)
	GetPeerIds(context.Context, *peerSearchOptions) ([]model.PeerIdentifier, error)
	CreatePeer(context.Context, *model.Peer) (*model.Peer, error)
	PrepareNewPeer(context.Context, model.InterfaceIdentifier) (*model.Peer, error)
	UpdatePeer(context.Context, *model.Peer) (*model.Peer, error)
	DeletePeer(context.Context, model.PeerIdentifier) error
	GetPeerQrCode(context.Context, *model.Peer) (io.Reader, error)
	GetPeerWgQuickConfig(context.Context, *model.Peer) (io.Reader, error)

	SendWgQuickConfigMail(context.Context, *mailOptions) error
}

type SortDirection int

const (
	SortAsc  SortDirection = 1
	SortDesc SortDirection = 0
)

const PageSizeAll = 0

// region user-options

type userSearchOptions struct {
	sortDirection SortDirection
	sortBy        string
	filter        string
	pageOffset    int
	pageSize      int
}

func UserSearchOptions() *userSearchOptions {
	return &userSearchOptions{
		sortDirection: SortAsc,
		sortBy:        "",
		filter:        "",
		pageOffset:    0,
		pageSize:      PageSizeAll,
	}
}

func (s *userSearchOptions) WithPageOffset(offset int) *userSearchOptions {
	s.pageOffset = offset
	return s
}

func (s *userSearchOptions) WithPageSize(size int) *userSearchOptions {
	s.pageSize = size
	return s
}

func (s *userSearchOptions) WithFilter(filter string) *userSearchOptions {
	s.filter = filter
	return s
}

func (s *userSearchOptions) WithSorting(property string, direction SortDirection) *userSearchOptions {
	s.sortBy = property
	s.sortDirection = direction
	return s
}

type userDeleteOptions struct {
	deletePeers bool
}

func UserDeleteOptions() *userDeleteOptions {
	return &userDeleteOptions{
		deletePeers: true,
	}
}

type userUpdateOptions struct {
	syncPeerState bool
}

func UserUpdateOptions() *userUpdateOptions {
	return &userUpdateOptions{
		syncPeerState: false,
	}
}

type userCreateOptions struct {
	createDefaultPeer     bool
	defaultPeerInterfaces []model.InterfaceIdentifier
}

func UserCreateOptions() *userCreateOptions {
	return &userCreateOptions{
		createDefaultPeer: false,
	}
}

func (s *userCreateOptions) WithDefaultPeer(createPeer bool, interfaces ...model.InterfaceIdentifier) *userCreateOptions {
	s.createDefaultPeer = createPeer
	s.defaultPeerInterfaces = interfaces
	return s
}

// endregion user-options

// region interface-options

type interfaceSearchOptions struct {
	sortDirection SortDirection
	sortBy        string
	filter        string
	typ           model.InterfaceType

	withStats bool
}

func InterfaceSearchOptions() *interfaceSearchOptions {
	return &interfaceSearchOptions{
		sortDirection: SortAsc,
		sortBy:        "",
		filter:        "",
		typ:           model.InterfaceTypeAny,
		withStats:     false,
	}
}

func (s *interfaceSearchOptions) WithFilter(filter string) *interfaceSearchOptions {
	s.filter = filter
	return s
}

func (s *interfaceSearchOptions) WithType(typ model.InterfaceType) *interfaceSearchOptions {
	s.typ = typ
	return s
}

func (s *interfaceSearchOptions) WithSorting(property string, direction SortDirection) *interfaceSearchOptions {
	s.sortBy = property
	s.sortDirection = direction
	return s
}

func (s *interfaceSearchOptions) WithStats(loadStats bool) *interfaceSearchOptions {
	s.withStats = loadStats
	return s
}

// endregion interface-options

// region import-options

type importOptions struct {
	// reserved for future use... (for example peer selection)
}

func ImportOptions() *importOptions {
	return nil
}

// endregion import-options

// region peer-options

type peerSearchOptions struct {
	sortDirection SortDirection
	sortBy        string
	pageOffset    int
	pageSize      int

	filter          string
	interfaceFilter model.InterfaceIdentifier
	userFilter      model.UserIdentifier

	withStats bool
}

func PeerSearchOptions() *peerSearchOptions {
	return &peerSearchOptions{
		sortDirection:   SortAsc,
		sortBy:          "",
		pageOffset:      0,
		pageSize:        PageSizeAll,
		filter:          "",
		interfaceFilter: "",
		userFilter:      "",
		withStats:       false,
	}
}

func (s *peerSearchOptions) WithPageOffset(offset int) *peerSearchOptions {
	s.pageOffset = offset
	return s
}

func (s *peerSearchOptions) WithPageSize(size int) *peerSearchOptions {
	s.pageSize = size
	return s
}

func (s *peerSearchOptions) WithSorting(property string, direction SortDirection) *peerSearchOptions {
	s.sortBy = property
	s.sortDirection = direction
	return s
}

func (s *peerSearchOptions) WithFilter(filter string) *peerSearchOptions {
	s.filter = filter
	return s
}

func (s *peerSearchOptions) WithInterface(interfaceId model.InterfaceIdentifier) *peerSearchOptions {
	s.interfaceFilter = interfaceId
	return s
}

func (s *peerSearchOptions) WithUser(userId model.UserIdentifier) *peerSearchOptions {
	s.userFilter = userId
	return s
}

func (s *peerSearchOptions) WithStats(loadStats bool) *peerSearchOptions {
	s.withStats = loadStats
	return s
}

// endregion peer-options

// region mail-options

type mailOptions struct {
	userFilter        model.UserIdentifier
	peerFilter        []model.PeerIdentifier
	includeAttachment bool
}

func MailOptions() *mailOptions {
	return &mailOptions{
		includeAttachment: true,
	}
}

func (s *mailOptions) WithUser(userId model.UserIdentifier) *mailOptions {
	s.userFilter = userId
	return s
}

func (s *mailOptions) WithPeers(peerIds ...model.PeerIdentifier) *mailOptions {
	s.peerFilter = peerIds
	return s
}

func (s *mailOptions) WithAttachment(useAttachment bool) *mailOptions {
	s.includeAttachment = useAttachment
	return s
}

// endregion mail-options
