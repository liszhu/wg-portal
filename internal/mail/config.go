package mail

type Encryption string

const (
	EncryptionNone     Encryption = "none"
	EncryptionTLS      Encryption = "tls"
	EncryptionStartTLS Encryption = "starttls"
)

type AuthType string

const (
	AuthPlain   AuthType = "plain"
	AuthLogin   AuthType = "login"
	AuthCramMD5 AuthType = "crammd5"
)

type Config struct {
	Host           string     `yaml:"host" envconfig:"EMAIL_HOST"`
	Port           int        `yaml:"port" envconfig:"EMAIL_PORT"`
	Encryption     Encryption `yaml:"encryption" envconfig:"EMAIL_ENCRYPTION"`
	CertValidation bool       `yaml:"cert_validation" envconfig:"EMAIL_CERT_VALIDATION"`
	Username       string     `yaml:"username" envconfig:"EMAIL_USERNAME"`
	Password       string     `yaml:"password" envconfig:"EMAIL_PASSWORD"`
	AuthType       AuthType   `yaml:"auth_type" envconfig:"EMAIL_AUTHTYPE"`

	From string `yaml:"from" envconfig:"EMAIL_FROM"`
}
