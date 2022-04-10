package ports

import (
	"context"

	"github.com/h44z/wg-portal/internal/domain"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type PlainAuthenticatorRepository interface {
	Config() domain.AuthenticatorConfig
	Authenticate(ctx context.Context, username, password string) error
	GetAllUserInfos(ctx context.Context) ([]domain.RawAuthenticatorUserInfo, error)

	GetUserInfo(ctx context.Context, username string) (domain.RawAuthenticatorUserInfo, error)
	ParseUserInfo(raw domain.RawAuthenticatorUserInfo) (*domain.AuthenticatorUserInfo, error)
}

type OauthAuthenticatorRepository interface {
	Config() domain.AuthenticatorConfig
	GetCallbackUrl() (string, error)
	AuthCodeUrl(state, nonce string) string
	Exchange(ctx context.Context, code string) (*domain.OAuthToken, error)

	GetUserInfo(ctx context.Context, token *domain.OAuthToken) (domain.RawAuthenticatorUserInfo, error)
	ParseUserInfo(raw domain.RawAuthenticatorUserInfo) (*domain.AuthenticatorUserInfo, error)
}

// NetlinkRepository provides low level functions for managing network interfaces.
type NetlinkRepository interface {
	Create(link *domain.NetLink) error
	Delete(link *domain.NetLink) error
	Get(name string) (*domain.NetLink, error)
	Up(link *domain.NetLink) error
	Down(link *domain.NetLink) error
	SetMTU(link *domain.NetLink, mtu int) error
	ReplaceAddr(link *domain.NetLink, addr *domain.LinkAddr) error
	AddAddr(link *domain.NetLink, addr *domain.LinkAddr) error
	ListAddr(link *domain.NetLink) ([]*domain.LinkAddr, error)
}

// WireGuardRepository provides low level functions for managing WireGuard interfaces.
type WireGuardRepository interface {
	Devices() ([]*wgtypes.Device, error)
	Device(name string) (*wgtypes.Device, error)
	ConfigureDevice(name string, cfg wgtypes.Config) error
}
