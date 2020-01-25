package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
)

type openbd []struct {
	Onix struct {
		RecordReference   string `json:"RecordReference"`
		NotificationType  string `json:"NotificationType"`
		ProductIdentifier struct {
			ProductIDType string `json:"ProductIDType"`
			IDValue       string `json:"IDValue"`
		} `json:"ProductIdentifier"`
		DescriptiveDetail struct {
			ProductComposition string `json:"ProductComposition"`
			ProductForm        string `json:"ProductForm"`
			ProductFormDetail  string `json:"ProductFormDetail"`
			TitleDetail        struct {
				TitleType    string `json:"TitleType"`
				TitleElement struct {
					TitleElementLevel string `json:"TitleElementLevel"`
					TitleText         struct {
						Collationkey string `json:"collationkey"`
						Content      string `json:"content"`
					} `json:"TitleText"`
					Subtitle struct {
						Collationkey string `json:"collationkey"`
						Content      string `json:"content"`
					} `json:"Subtitle"`
				} `json:"TitleElement"`
			} `json:"TitleDetail"`
			Contributor []struct {
				SequenceNumber  string   `json:"SequenceNumber"`
				ContributorRole []string `json:"ContributorRole"`
				PersonName      struct {
					Collationkey string `json:"collationkey"`
					Content      string `json:"content"`
				} `json:"PersonName"`
				BiographicalNote string `json:"BiographicalNote,omitempty"`
			} `json:"Contributor"`
			Language []struct {
				LanguageRole string `json:"LanguageRole"`
				LanguageCode string `json:"LanguageCode"`
				CountryCode  string `json:"CountryCode"`
			} `json:"Language"`
			Extent []struct {
				ExtentType  string `json:"ExtentType"`
				ExtentValue string `json:"ExtentValue"`
				ExtentUnit  string `json:"ExtentUnit"`
			} `json:"Extent"`
			Subject []struct {
				SubjectSchemeIdentifier string `json:"SubjectSchemeIdentifier"`
				SubjectCode             string `json:"SubjectCode"`
			} `json:"Subject"`
		} `json:"DescriptiveDetail"`
		CollateralDetail struct {
			TextContent []struct {
				TextType        string `json:"TextType"`
				ContentAudience string `json:"ContentAudience"`
				Text            string `json:"Text"`
			} `json:"TextContent"`
			SupportingResource []struct {
				ResourceContentType string `json:"ResourceContentType"`
				ContentAudience     string `json:"ContentAudience"`
				ResourceMode        string `json:"ResourceMode"`
				ResourceVersion     []struct {
					ResourceForm           string `json:"ResourceForm"`
					ResourceVersionFeature []struct {
						ResourceVersionFeatureType string `json:"ResourceVersionFeatureType"`
						FeatureValue               string `json:"FeatureValue"`
					} `json:"ResourceVersionFeature"`
					ResourceLink string `json:"ResourceLink"`
				} `json:"ResourceVersion"`
			} `json:"SupportingResource"`
		} `json:"CollateralDetail"`
		PublishingDetail struct {
			Imprint struct {
				ImprintIdentifier []struct {
					ImprintIDType string `json:"ImprintIDType"`
					IDValue       string `json:"IDValue"`
				} `json:"ImprintIdentifier"`
				ImprintName string `json:"ImprintName"`
			} `json:"Imprint"`
			PublishingDate []struct {
				PublishingDateRole string `json:"PublishingDateRole"`
				Date               string `json:"Date"`
			} `json:"PublishingDate"`
		} `json:"PublishingDetail"`
		ProductSupply struct {
			SupplyDetail struct {
				ReturnsConditions struct {
					ReturnsCodeType string `json:"ReturnsCodeType"`
					ReturnsCode     string `json:"ReturnsCode"`
				} `json:"ReturnsConditions"`
				ProductAvailability string `json:"ProductAvailability"`
				Price               []struct {
					PriceType    string `json:"PriceType"`
					PriceAmount  string `json:"PriceAmount"`
					CurrencyCode string `json:"CurrencyCode"`
				} `json:"Price"`
			} `json:"SupplyDetail"`
		} `json:"ProductSupply"`
	} `json:"onix"`
	Hanmoto struct {
		Toji         string `json:"toji"`
		Zaiko        int    `json:"zaiko"`
		Maegakinado  string `json:"maegakinado"`
		Kaisetsu105W string `json:"kaisetsu105w"`
		Tsuiki       string `json:"tsuiki"`
		Genrecodetrc int    `json:"genrecodetrc"`
		Kankoukeitai string `json:"kankoukeitai"`
		Jyuhan       []struct {
			Date    string `json:"date"`
			Ctime   string `json:"ctime"`
			Suri    int    `json:"suri"`
			Comment string `json:"comment,omitempty"`
		} `json:"jyuhan"`
		Hastameshiyomi bool `json:"hastameshiyomi"`
		Author         []struct {
			Listseq     int    `json:"listseq"`
			Dokujikubun string `json:"dokujikubun"`
		} `json:"author"`
		Datemodified string `json:"datemodified"`
		Datecreated  string `json:"datecreated"`
		Reviews      []struct {
			PostUser   string `json:"post_user"`
			Reviewer   string `json:"reviewer"`
			SourceID   int    `json:"source_id"`
			KubunID    int    `json:"kubun_id"`
			Source     string `json:"source"`
			Choyukan   string `json:"choyukan"`
			Han        string `json:"han"`
			Link       string `json:"link"`
			Appearance string `json:"appearance"`
			Gou        string `json:"gou"`
		} `json:"reviews"`
		Hanmotoinfo struct {
			Name     string `json:"name"`
			Yomi     string `json:"yomi"`
			URL      string `json:"url"`
			Twitter  string `json:"twitter"`
			Facebook string `json:"facebook"`
		} `json:"hanmotoinfo"`
		Dateshuppan string `json:"dateshuppan"`
	} `json:"hanmoto"`
	Summary struct {
		Isbn      string `json:"isbn"`
		Title     string `json:"title"`
		Volume    string `json:"volume"`
		Series    string `json:"series"`
		Publisher string `json:"publisher"`
		Pubdate   string `json:"pubdate"`
		Cover     string `json:"cover"`
		Author    string `json:"author"`
	} `json:"summary"`
}

type openbdAPI struct {
	OpenBD    openbd
	data      []byte
	Title     string
	Author    string
	Publisher string
	Pubdate   string
	ISBN      string
}

func (bd *openbdAPI) Get(isbn string) error {
	resp, err := http.Get("https://api.openbd.jp/v1/get?pretty&isbn=" + isbn)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bd.data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("OpenBD response: " + resp.Status)
	}
	return bd.parse()
}
func (bd *openbdAPI) Save(path string) error {
	json := filepath.Join(path, "isbn_openbd.json")
	return ioutil.WriteFile(json, bd.data, 0644)
}
func (bd *openbdAPI) Load(path string) (err error) {
	json := filepath.Join(path, "isbn_openbd.json")
	bd.data, err = ioutil.ReadFile(json)
	if err != nil {
		return
	}
	err = bd.parse()
	return
}

func (bd *openbdAPI) parse() error {
	if err := json.Unmarshal(bd.data, &bd.OpenBD); err != nil {
		return err
	}
	for _, item := range bd.OpenBD {
		if item.Summary.Author == "" || item.Summary.Title == "" {
			continue
		}
		bd.Title = item.Summary.Title
		bd.Publisher = item.Summary.Publisher
		if item.Summary.Volume != "" {
			bd.Title += " " + item.Summary.Volume
		}
		var a []string
		for _, con := range item.Onix.DescriptiveDetail.Contributor {
			a = append(a, con.PersonName.Content)
		}
		bd.Author = strings.ReplaceAll(strings.Join(a, "／"), " ", "")
		if bd.Author == "" {
			bd.Author = item.Summary.Author
		}

		bd.Pubdate = item.Summary.Pubdate
		bd.ISBN = item.Summary.Isbn
		return nil
	}
	//title,authorが空白ならエラー
	return errors.New("opendbAPI unknown format")
}
