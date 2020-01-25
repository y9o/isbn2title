package main

import (
	"encoding/xml"
	"errors"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

type kokkaibd struct {
	XMLName    xml.Name `xml:"rss"`
	Text       string   `xml:",chardata"`
	Dc         string   `xml:"dc,attr"`
	Dcmitype   string   `xml:"dcmitype,attr"`
	Version    string   `xml:"version,attr"`
	Xsi        string   `xml:"xsi,attr"`
	Rdfs       string   `xml:"rdfs,attr"`
	Dcterms    string   `xml:"dcterms,attr"`
	Dcndl      string   `xml:"dcndl,attr"`
	OpenSearch string   `xml:"openSearch,attr"`
	Rdf        string   `xml:"rdf,attr"`
	Channel    struct {
		Text         string `xml:",chardata"`
		Title        string `xml:"title"`
		Link         string `xml:"link"`
		Description  string `xml:"description"`
		Language     string `xml:"language"`
		TotalResults string `xml:"totalResults"`
		StartIndex   string `xml:"startIndex"`
		ItemsPerPage string `xml:"itemsPerPage"`
		Item         []struct {
			Text        string `xml:",chardata"`
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			Author      string `xml:"author"`
			Category    string `xml:"category"`
			Guid        struct {
				Text        string `xml:",chardata"`
				IsPermaLink string `xml:"isPermaLink,attr"`
			} `xml:"guid"`
			PubDate              string `xml:"pubDate"`
			TitleTranscription   string `xml:"titleTranscription"`
			Creator              string `xml:"creator"`
			CreatorTranscription string `xml:"creatorTranscription"`
			Publisher            string `xml:"publisher"`
			Issued               []struct {
				Text string `xml:",chardata"`
				Type string `xml:"type,attr"`
			} `xml:"issued"`
			Extent     []string `xml:"extent"`
			Identifier []struct {
				Text string `xml:",chardata"`
				Type string `xml:"type,attr"`
			} `xml:"identifier"`
			Subject []struct {
				Text string `xml:",chardata"`
				Type string `xml:"type,attr"`
			} `xml:"subject"`
			SeeAlso []struct {
				Text     string `xml:",chardata"`
				Resource string `xml:"resource,attr"`
			} `xml:"seeAlso"`
			SeriesTitle string `xml:"seriesTitle"`
			IsPartOf    struct {
				Text     string `xml:",chardata"`
				Resource string `xml:"resource,attr"`
			} `xml:"isPartOf"`
			SeriesTitleTranscription string `xml:"seriesTitleTranscription"`
			Price                    string `xml:"price"`
			Volume                   string `xml:"volume"`
		} `xml:"item"`
	} `xml:"channel"`
}

type kokkaiAPI struct {
	Kokkai    kokkaibd
	data      []byte
	Title     string
	Author    string
	Publisher string
	Pubdate   string
	ISBN      string
}

func (bd *kokkaiAPI) Get(isbn string) error {
	resp, err := http.Get("http://iss.ndl.go.jp/api/opensearch?isbn=" + isbn)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bd.data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("Response: " + resp.Status)
	}
	return bd.parse()
}
func (bd *kokkaiAPI) Save(path string) error {
	json := filepath.Join(path, "isbn_kokkai.xml")
	return ioutil.WriteFile(json, bd.data, 0644)
}
func (bd *kokkaiAPI) Load(path string) (err error) {
	json := filepath.Join(path, "isbn_kokkai.xml")
	bd.data, err = ioutil.ReadFile(json)
	if err != nil {
		return
	}
	err = bd.parse()
	return
}

func (bd *kokkaiAPI) parse() error {
	if err := xml.Unmarshal(bd.data, &bd.Kokkai); err != nil {
		return err
	}
	for _, item := range bd.Kokkai.Channel.Item {
		if item.Author == "" {
			continue
		}
		if item.Title == "" {
			continue
		}
		//カセットテープなどを除外 必要?
		if item.Category != "本" {
			continue
		}
		bd.Author = item.Author
		bd.Title = item.Title
		bd.Publisher = item.Publisher
		for _, isbn := range item.Identifier {
			if isbn.Type == "dcndl:ISBN" && len(bd.ISBN) != 13 {
				bd.ISBN = isbn.Text
			}
		}
		for _, v := range item.Issued {
			if v.Type == "dcterms:W3CDTF" {
				bd.Pubdate = v.Text
			}
		}
		if item.Volume != "" {
			bd.Title += " " + item.Volume
		}
		return nil
	}
	//title,authorが空白ならエラー
	return errors.New("kokkaiAPI unknown format")
}
