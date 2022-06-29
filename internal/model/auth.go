package model

type LoginProvider string

type LoginProviderInfo struct {
	Identifier  string
	Name        string
	ProviderUrl string
	CallbackUrl string
}
