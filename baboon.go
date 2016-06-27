package baboon

import (
	"encoding/base64"
	"log"
	"net/mail"
	"sync"

	"github.com/aymerick/raymond"
	"github.com/keighl/mandrill"
	"github.com/mattbaird/gochimp"
	"github.com/sendgrid/sendgrid-go"
)

type Client struct {
	sgClient *sendgrid.SGClient
	chimpAPI *gochimp.ChimpAPI

	tmpls map[string]string

	initMutex sync.RWMutex

	err chan error
}

func New(mailchimpKey, sendGridUser, sendGridPw string) *Client {
	var client Client
	client.sgClient = sendgrid.NewSendGridClient(sendGridUser, sendGridPw)
	client.chimpAPI = gochimp.NewChimp(mailchimpKey, true)
	client.tmpls = map[string]string{}

	client.initMutex.Lock()
	client.err = make(chan error)
	go func() {
		defer client.initMutex.Unlock()

		chimpList, err := client.chimpAPI.TemplatesList(gochimp.TemplatesList{
			Types:   gochimp.TemplateListType{User: true, Gallery: true, Base: true},
			Filters: gochimp.TemplateListFilter{IncludeDragAndDrop: true},
		})
		if err != nil {
			client.err <- err
			return
		}

		for _, tmpl := range chimpList.User {
			info, err := client.chimpAPI.TemplatesInfo(gochimp.TemplateInfo{
				TemplateID: tmpl.Id,
				Type:       "user",
			})
			if err != nil {
				client.err <- err
				return
			}
			client.tmpls[tmpl.Name] = info.Source
		}

		close(client.err)
	}()

	return &client
}

func (c *Client) WaitInitDone() error {
	return <-c.err
}

var DsiableSending bool

func (c *Client) MessagesSendTemplate(msg *mandrill.Message, name string, contents interface{}) ([]*mandrill.Response, error) {
	log.Println("[baboon] sending by SendGrid")
	if DsiableSending {
		log.Println("[baboon] sending is disabled.")
		return nil, nil
	}

	c.initMutex.RLock()
	defer c.initMutex.RUnlock()

	data := map[string]interface{}{}
	for _, mvar := range msg.GlobalMergeVars {
		data[mvar.Name] = mvar.Content
	}
	result, err := raymond.Render(c.tmpls[name], data)
	if err != nil {
		return nil, err
	}

	m := sendgrid.NewMail()
	m.SetHTML(result)
	for _, to := range msg.To {
		m.AddTo(to.Email)
	}
	m.SetFromEmail(&mail.Address{Name: msg.FromName, Address: msg.FromEmail})
	m.SetFromName(msg.FromName)
	m.SetSubject(msg.Subject)

	for _, att := range msg.Attachments {
		cnt, err := base64.StdEncoding.DecodeString(att.Content)
		if err != nil {
			return nil, err
		}
		m.AddAttachmentFromStream(att.Name, string(cnt))
	}

	if err := c.sgClient.Send(m); err != nil {
		return nil, err
	}

	return nil, nil
}
