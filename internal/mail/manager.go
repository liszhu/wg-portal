package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/h44z/wg-portal/internal/model"
	"github.com/pkg/errors"
	mail "github.com/xhit/go-simple-mail/v2"
)

type Manager interface {
	Send(ctx context.Context, subject, body string, to []string, options *Options) error
	SendConfigWithLink(ctx context.Context, user *model.User, peer *model.Peer, link string) error
	SendConfigWithAttachment(ctx context.Context, user *model.User, peer *model.Peer, qr, cfg io.Reader) error
}

type Options struct {
	ReplyTo     string // defaults to the sender
	HtmlBody    string // if html body is empty, a text-only email will be sent
	Cc          []string
	Bcc         []string
	Attachments []Attachment
}

type Attachment struct {
	Name        string
	ContentType string
	Data        io.Reader
	Embedded    bool
}

type smtpManager struct {
	cfg *Config
	tpl *templateHandler
}

func NewSmtpManager(cfg *Config) (*smtpManager, error) {
	tpl, err := newTemplateHandler()
	if err != nil {
		return nil, err
	}
	return &smtpManager{tpl: tpl, cfg: cfg}, nil
}

// Send sends a mail.
func (r *smtpManager) Send(_ context.Context, subject, body string, to []string, options *Options) error {
	if options == nil {
		options = &Options{}
	}
	r.setDefaultOptions(r.cfg.From, options)

	if len(to) == 0 {
		return errors.New("missing email recipient")
	}

	uniqueTo := r.uniqueAddresses(to)
	email := mail.NewMSG()
	email.SetFrom(r.cfg.From).
		AddTo(uniqueTo...).
		SetReplyTo(options.ReplyTo).
		SetSubject(subject).
		SetBody(mail.TextPlain, body)

	if len(options.Cc) > 0 {
		// the underlying mail library does not allow the same address to appear in TO and CC... so filter entries that are already included
		// in the TO addresses
		cc := r.removeDuplicates(r.uniqueAddresses(options.Cc), uniqueTo)
		email.AddCc(cc...)
	}
	if len(options.Bcc) > 0 {
		// the underlying mail library does not allow the same address to appear in TO or CC and BCC... so filter entries that are already
		// included in the TO and CC addresses
		bcc := r.removeDuplicates(r.uniqueAddresses(options.Bcc), uniqueTo)
		bcc = r.removeDuplicates(bcc, options.Cc)

		email.AddCc(r.uniqueAddresses(options.Bcc)...)
	}
	if options.HtmlBody != "" {
		email.AddAlternative(mail.TextHTML, options.HtmlBody)
	}

	for _, attachment := range options.Attachments {
		attachmentData, err := ioutil.ReadAll(attachment.Data)
		if err != nil {
			return fmt.Errorf("failed to read attachment data for %s: %w", attachment.Name, err)
		}

		if attachment.Embedded {
			email.AddInlineData(attachmentData, attachment.Name, attachment.ContentType)
		} else {
			email.AddAttachmentData(attachmentData, attachment.Name, attachment.ContentType)
		}
	}

	// Call Send and pass the client
	srv := r.getMailServer()
	client, err := srv.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	err = email.Send(client)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (r *smtpManager) SendConfigWithLink(ctx context.Context, user *model.User, peer *model.Peer, link string) error {
	//TODO implement me
	panic("implement me")
}

func (r *smtpManager) SendConfigWithAttachment(ctx context.Context, user *model.User, peer *model.Peer, qr, cfg io.Reader) error {
	//TODO implement me
	panic("implement me")
}

func (r *smtpManager) setDefaultOptions(sender string, options *Options) {
	if options.ReplyTo == "" {
		options.ReplyTo = sender
	}
}

func (r *smtpManager) getMailServer() *mail.SMTPServer {
	srv := mail.NewSMTPClient()

	srv.ConnectTimeout = 30 * time.Second
	srv.SendTimeout = 30 * time.Second
	srv.Host = r.cfg.Host
	srv.Port = r.cfg.Port
	srv.Username = r.cfg.Username
	srv.Password = r.cfg.Password

	switch r.cfg.Encryption {
	case EncryptionTLS:
		srv.Encryption = mail.EncryptionSSLTLS
	case EncryptionStartTLS:
		srv.Encryption = mail.EncryptionSTARTTLS
	default: // MailEncryptionNone
		srv.Encryption = mail.EncryptionNone
	}
	srv.TLSConfig = &tls.Config{ServerName: srv.Host, InsecureSkipVerify: !r.cfg.CertValidation}
	switch r.cfg.AuthType {
	case AuthPlain:
		srv.Authentication = mail.AuthPlain
	case AuthLogin:
		srv.Authentication = mail.AuthLogin
	case AuthCramMD5:
		srv.Authentication = mail.AuthCRAMMD5
	}

	return srv
}

// uniqueAddresses removes duplicates in the given string slice
func (r *smtpManager) uniqueAddresses(slice []string) []string {
	keys := make(map[string]struct{})
	uniqueSlice := make([]string, 0, len(slice))
	for _, entry := range slice {
		if _, exists := keys[entry]; !exists {
			keys[entry] = struct{}{}
			uniqueSlice = append(uniqueSlice, entry)
		}
	}
	return uniqueSlice
}

func (r *smtpManager) removeDuplicates(slice []string, remove []string) []string {
	uniqueSlice := make([]string, 0, len(slice))

	for _, i := range remove {
		for _, j := range slice {
			if i != j {
				uniqueSlice = append(uniqueSlice, j)
			}
		}
	}
	return uniqueSlice
}
