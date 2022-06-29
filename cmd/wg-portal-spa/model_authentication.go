package main

type OauthInitiationResponse struct {
	RedirectUrl string
	State       string
}

type SessionInfoResponse struct {
	LoggedIn       bool
	IsAdmin        bool    `json:"IsAdmin,omitempty"`
	UserIdentifier *string `json:"UserIdentifier,omitempty"`
}
