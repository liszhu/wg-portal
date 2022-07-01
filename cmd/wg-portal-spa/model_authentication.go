package main

type OauthInitiationResponse struct {
	RedirectUrl string
	State       string
}

type SessionInfoResponse struct {
	LoggedIn       bool
	IsAdmin        bool    `json:"IsAdmin,omitempty"`
	UserIdentifier *string `json:"UserIdentifier,omitempty"`
	UserFirstname  *string `json:"UserFirstname,omitempty"`
	UserLastname   *string `json:"UserLastname,omitempty"`
	UserEmail      *string `json:"UserEmail,omitempty"`
}
