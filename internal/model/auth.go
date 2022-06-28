package model

type LoginProvider string

type LoginProviderInfo struct {
	ID          string
	Name        string
	ProviderUrl string
	CallbackUrl string
}
