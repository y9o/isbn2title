package main

import (
	"log"
	"strings"
	"testing"
)

func TestTemplate(t *testing.T) {

	apis := make([]isbnAPI, 0, 4)
	for _, apiname := range strings.Split("google,openbd,kokkai,calilWEB", ",") {
		switch strings.ToLower(apiname) {
		case "openbd":
			apis = append(apis, &openbdAPI{})
		case "google":
			apis = append(apis, &googleAPI{})
		case "kokkai":
			apis = append(apis, &kokkaiAPI{})
		default:
			api, err := NewWebSite(apiname + ".yml")
			if err != nil {
				t.Errorf("err(%s)%s\n", apiname, err)
			} else {
				apis = append(apis, api)
			}
		}
	}
	for _, api := range apis {
		api.Load("test")
		newname := makeFileNameFromBD(api, &option{rename: "[{{.Author}}] {{.Title}} {{with .Publisher}}[{{.}}]{{end}}{{with .Pubdate}}[{{.}}]{{end}}[ISBN {{.ISBN}}]{{if hasField . \"Google\"}}{{$v := index .Google.Items 0}} G:{{$v.VolumeInfo.PublishedDate}} {{end}}"})
		log.Printf("test(%T): => \"%s\"\n", api, newname)
	}
}
