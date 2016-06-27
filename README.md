# baboon

Send Mandrill Transaction Emails Over SendGrid. For development/demo.

## How baboon works

Requirements:

1. Create a development SendGrid, Mailchimp account;
2. Sync your production templates to demo mailchimp accounts;
3. Replace your mailchimp client with an interface with methods;
4. Package dependency:
	```
	github.com/mattbaird/gochimp
	github.com/sendgrid/sendgrid-go
	github.com/aymerick/raymond
	github.com/keighl/mandrill
	```
5. Use __Handlebar__ as your Mailchimp template language;
6. For transactional emails (Mandrill);

### Synchimp Usage

create `synchimp.json`

```json
{
	"demos": [
		{"key": "g_D87PYvYzc9!ux8C.uDrA9JerEnV!ND-us51"}
	],
	"templates": {
		"mailchimp-template-name-1": "mandrill-slug-name-1",
		"mailchimp-template-name-2": "mandrill-slug-name-2"
	}
}
```

```bash
go get github.com/theplant/baboon/synchimp

PROD_KEY=hpsBYnq.SufygJ22#5733UkKPrQ93GpG-us51 synchimp -cfg synchimp.json
```

### Baboon Usage

Baboon only implement Mailchimp's MessagesSendTemplate

```go
// instead of using package gochimp directly, define an interface

var MandrillClient interface {
	MessagesSendTemplate(*mandrill.Message, string, interface{}) ([]*mandrill.Response, error)
}

if isProd {
	// init with gochimp
	MandrillClient = gochimp.New("key", true)
	return
}

MandrillClient = baboon.New("MailChimpAPIKey", "SendGrid.User", "SendGrid.Pw")

go func() {
	start := time.Now()
	if err := MandrillClient.(*baboon.Client).WaitInitDone(); err != nil {
		panic(err)
	}
	baboon.DsiableSending = !Cfg.EnableBaboon
	log.Printf("[baboon] init done (took %s)\n", time.Now().Sub(start))
}()

MandrillClient.MessagesSendTemplate(msg, slug, nil)
```
