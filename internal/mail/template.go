package mail

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	htmlTemplate "html/template"
	"io"

	"github.com/h44z/wg-portal/internal/model"
	"github.com/pkg/errors"
)

//go:embed tpl_files/*
var TemplateFiles embed.FS

type templateHandler struct {
	htmlTemplates *htmlTemplate.Template
	templates     *template.Template
}

func newTemplateHandler() (*templateHandler, error) {
	htmlTemplateCache, err := htmlTemplate.New("WireGuard").ParseFS(TemplateFiles, "tpl_files/*.gohtml")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse html template files")
	}

	templateCache, err := template.New("WireGuard").ParseFS(TemplateFiles, "tpl_files/*.gotpl")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse text template files")
	}

	handler := &templateHandler{
		htmlTemplates: htmlTemplateCache,
		templates:     templateCache,
	}

	return handler, nil
}

func (c templateHandler) GetConfigMail(user *model.User, peer *model.Peer, link string) (io.Reader, io.Reader, error) {
	var tplBuff bytes.Buffer
	var htmlTplBuff bytes.Buffer

	err := c.templates.ExecuteTemplate(&tplBuff, "mail_with_link.gotpl", map[string]interface{}{
		"User": user,
		"Peer": peer,
		"Link": link,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute template mail_with_link.gotpl")
	}

	err = c.htmlTemplates.ExecuteTemplate(&tplBuff, "mail_with_link.gohtml", map[string]interface{}{
		"User": user,
		"Peer": peer,
		"Link": link,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute template mail_with_link.gohtml")
	}

	return &tplBuff, &htmlTplBuff, nil
}

func (c templateHandler) GetConfigMailWithAttachment(user *model.User, peer *model.Peer) (io.Reader, io.Reader, error) {
	var tplBuff bytes.Buffer
	var htmlTplBuff bytes.Buffer

	err := c.templates.ExecuteTemplate(&tplBuff, "mail_with_attachment.gotpl", map[string]interface{}{
		"User": user,
		"Peer": peer,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute template mail_with_attachment.gotpl")
	}

	err = c.htmlTemplates.ExecuteTemplate(&tplBuff, "mail_with_attachment.gohtml", map[string]interface{}{
		"User": user,
		"Peer": peer,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute template mail_with_attachment.gohtml")
	}

	return &tplBuff, &htmlTplBuff, nil
}
