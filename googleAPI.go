package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
)

type googlebd struct {
	Kind       string `json:"kind"`
	TotalItems int    `json:"totalItems"`
	Items      []struct {
		Kind       string `json:"kind"`
		ID         string `json:"id"`
		Etag       string `json:"etag"`
		SelfLink   string `json:"selfLink"`
		VolumeInfo struct {
			Title               string   `json:"title"`
			Subtitle            string   `json:"subtitle"`
			Authors             []string `json:"authors"`
			PublishedDate       string   `json:"publishedDate"`
			Description         string   `json:"description"`
			IndustryIdentifiers []struct {
				Type       string `json:"type"`
				Identifier string `json:"identifier"`
			} `json:"industryIdentifiers"`
			ReadingModes struct {
				Text  bool `json:"text"`
				Image bool `json:"image"`
			} `json:"readingModes"`
			PageCount        int    `json:"pageCount"`
			PrintType        string `json:"printType"`
			MaturityRating   string `json:"maturityRating"`
			AllowAnonLogging bool   `json:"allowAnonLogging"`
			ContentVersion   string `json:"contentVersion"`
			ImageLinks       struct {
				SmallThumbnail string `json:"smallThumbnail"`
				Thumbnail      string `json:"thumbnail"`
			} `json:"imageLinks"`
			Language            string `json:"language"`
			PreviewLink         string `json:"previewLink"`
			InfoLink            string `json:"infoLink"`
			CanonicalVolumeLink string `json:"canonicalVolumeLink"`
		} `json:"volumeInfo"`
		SaleInfo struct {
			Country     string `json:"country"`
			Saleability string `json:"saleability"`
			IsEbook     bool   `json:"isEbook"`
		} `json:"saleInfo"`
		AccessInfo struct {
			Country                string `json:"country"`
			Viewability            string `json:"viewability"`
			Embeddable             bool   `json:"embeddable"`
			PublicDomain           bool   `json:"publicDomain"`
			TextToSpeechPermission string `json:"textToSpeechPermission"`
			Epub                   struct {
				IsAvailable bool `json:"isAvailable"`
			} `json:"epub"`
			Pdf struct {
				IsAvailable bool `json:"isAvailable"`
			} `json:"pdf"`
			WebReaderLink       string `json:"webReaderLink"`
			AccessViewStatus    string `json:"accessViewStatus"`
			QuoteSharingAllowed bool   `json:"quoteSharingAllowed"`
		} `json:"accessInfo"`
		SearchInfo struct {
			TextSnippet string `json:"textSnippet"`
		} `json:"searchInfo"`
	} `json:"items"`
}

type googleAPI struct {
	Google    googlebd
	data      []byte
	Title     string
	Author    string
	Publisher string
	Pubdate   string
	ISBN      string
}

func (bd *googleAPI) Get(isbn string) error {
	resp, err := http.Get("https://www.googleapis.com/books/v1/volumes?q=isbn:" + isbn)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bd.data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("Google response: " + resp.Status)
	}
	return bd.parse()
}
func (bd *googleAPI) Save(path string) error {
	json := filepath.Join(path, "isbn_google.json")
	return ioutil.WriteFile(json, bd.data, 0644)
}
func (bd *googleAPI) Load(path string) (err error) {
	json := filepath.Join(path, "isbn_google.json")
	bd.data, err = ioutil.ReadFile(json)
	if err != nil {
		return
	}
	err = bd.parse()
	return
}

func (bd *googleAPI) parse() error {
	if err := json.Unmarshal(bd.data, &bd.Google); err != nil {
		return err
	}
	//title,authorが空白ならエラー
	if bd.Google.TotalItems < 1 || bd.Google.Items[0].VolumeInfo.Title == "" || len(bd.Google.Items[0].VolumeInfo.Authors) < 1 || bd.Google.Items[0].VolumeInfo.Authors[0] == "" {
		return errors.New("googlebd unknown format")
	}
	for _, item := range bd.Google.Items {
		if len(item.VolumeInfo.Authors) == 0 || item.VolumeInfo.Title == "" {
			continue
		}
		bd.Title = item.VolumeInfo.Title
		if item.VolumeInfo.Subtitle != "" {
			bd.Title += " " + item.VolumeInfo.Subtitle
		}
		bd.Author = strings.ReplaceAll(strings.Join(bd.Google.Items[0].VolumeInfo.Authors, "／"), " ", "")
		bd.Pubdate = item.VolumeInfo.PublishedDate
		for _, isbn := range item.VolumeInfo.IndustryIdentifiers {
			if bd.ISBN == "" || isbn.Type == "ISBN_13" {
				bd.ISBN = isbn.Identifier
			}
		}
	}
	return nil
}
