package config

type WebConfig struct {
	ExternalUrl      string `yaml:"external_url"`
	ListeningAddress string `yaml:"listening_address"`
	SessionSecret    string `yaml:"session_secret"`
	SiteTitle        string `yaml:"site_title"`
	SiteCompanyName  string `yaml:"site_company_name"`
}
