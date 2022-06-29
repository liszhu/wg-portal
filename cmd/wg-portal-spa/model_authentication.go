package main

type OauthInitiationResponse struct {
	RedirectUrl string
	State       string
}

type SessionInfoResponse struct {
	LoggedIn bool
	IsAdmin  bool    `json:"IsAdmin,omitempty"`
	UserId   *string `json:"UserId,omitempty"`
}
