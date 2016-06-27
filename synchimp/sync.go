// syncchimp syncs templats from mailchimp account to our test/dev mandrill accounts.
//
// Production templates could updated by providing two environment variables PROD_KEY.
//
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/bom-d-van/sidekick"
	"github.com/mattbaird/gochimp"
)

var config = struct {
	Demos     []*Account
	Templates map[string]string
}{
	Templates: map[string]string{},
}

type Account struct {
	Key string
	*gochimp.ChimpAPI
	list gochimp.TemplatesListResponse
}

func (a *Account) init() error {
	a.ChimpAPI = gochimp.NewChimp(a.Key, true)
	var err error
	if a.list, err = a.TemplatesList(gochimp.TemplatesList{
		Types:   gochimp.TemplateListType{User: true, Gallery: true, Base: true},
		Filters: gochimp.TemplateListFilter{IncludeDragAndDrop: true},
	}); err != nil {
		return err
	}
	return nil
}

// usage: PROD_KEY="key" go run scripts/sync_mailchimp_templates.go
//
// production PROD_KEY could be found in provision repository.
func main() {
	cfgname := flag.String("cfg", "synchimp.json", "synchimp config file")
	if *cfgname == "" {
		fmt.Println("please specify cfg file with -cfg")
		os.Exit(1)
	}
	cfgf, err := os.Open(*cfgname)
	if err != nil {
		fmt.Printf("failed to open %s: %s\n", *cfgname, err)
		os.Exit(1)
	}
	if err := json.NewDecoder(cfgf).Decode(&config); err != nil {
		fmt.Printf("failed to decode %s: %s\n", *cfgname, err)
		os.Exit(1)
	}

	start := time.Now()
	key := os.Getenv("PROD_KEY")
	if key == "" {
		fmt.Println("Please specify source/production mailchimp account key with env PROD_KEY")
		os.Exit(1)
	}
	srcChimp := gochimp.NewChimp(key, true)

	for _, demo := range config.Demos {
		if err := demo.init(); err != nil {
			fmt.Printf("failed to init demo account (key=%s): %s", demo.Key, err)
			os.Exit(1)
		}
	}

	srcList, err := srcChimp.TemplatesList(gochimp.TemplatesList{
		Types:   gochimp.TemplateListType{User: true, Gallery: true, Base: true},
		Filters: gochimp.TemplateListFilter{IncludeDragAndDrop: true},
	})
	if err != nil {
		panic(err)
	}

	for _, tmpl := range srcList.User {
		slug, ok := config.Templates[tmpl.Name]
		if !ok {
			continue
		}
		fmt.Println("slug:", tmpl.Name)

		info, err := srcChimp.TemplatesInfo(gochimp.TemplateInfo{
			TemplateID: tmpl.Id,
			Type:       "user",
		})
		if err != nil {
			panic(err)
		}

		for _, demo := range config.Demos {
			fmt.Printf("	demo: %s\n", demo.Key)
			if oldTmpl := doesTemplateExist(demo.list.User, slug); oldTmpl != nil {
				_, err := demo.TemplatesUpdate(gochimp.TemplatesUpdate{
					TemplateID: oldTmpl.Id,
					Values: gochimp.TemplatesUpdateValues{
						Name: slug,
						HTML: info.Source,
					},
				})
				if err != nil {
					fmt.Println(err)
				}
			} else {
				_, err := demo.TemplatesAdd(gochimp.TemplatesAdd{
					Name: slug,
					HTML: info.Source,
				})
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}

	fmt.Println("Took", time.Now().Sub(start))
}

func doesTemplateExist(tmpls []gochimp.UserTemplate, name string) *gochimp.UserTemplate {
	for _, tmpl := range tmpls {
		if tmpl.Name == name {
			return &tmpl
		}
	}
	return nil
}
