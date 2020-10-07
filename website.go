package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/antchfx/htmlquery"
	"gopkg.in/yaml.v2"
)

type siteRegexp struct {
	Pattern string `yaml:"Pattern"`
	Repl    string `yaml:"Replace,omitempty"`
}

func (r *siteRegexp) Replace(str string) string {
	if r.Pattern == "" {
		return str
	}
	pattern := regexp.MustCompile(r.Pattern)
	return pattern.ReplaceAllString(str, r.Repl)
}

type siteItem struct {
	XPath  []string   `yaml:"XPath"`
	Join   string     `yaml:"Join,omitempty"`
	Regexp siteRegexp `yaml:"Regexp,omitempty"`
}
type parseSite struct {
	URL       string `yaml:"URL"`
	UserAgent string `yaml:"UA"`
	File      string `yaml:"File"`
	Parse     struct {
		Author    siteItem `yaml:"Author"`
		Title     siteItem `yaml:"Title"`
		Publisher siteItem `yaml:"Publisher"`
		Pubdate   siteItem `yaml:"Pubdate"`
		ISBN      siteItem `yaml:"ISBN"`
	} `yaml:"Parse"`
}

type webSite struct {
	file      string
	web       parseSite
	data      []byte
	Title     string
	Author    string
	Publisher string
	Pubdate   string
	ISBN      string
}

func NewWebSite(file string) (*webSite, error) {
	ret := &webSite{file: file}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &ret.web)
	if err != nil {
		return nil, err
	}
	if ret.web.Parse.Author.Regexp.Pattern != "" {
		_, err := regexp.Compile(ret.web.Parse.Author.Regexp.Pattern)
		if err != nil {
			return nil, fmt.Errorf("Parse.Author.Regexp.Patternを見直して下さい %w", err)
		}
	}
	if ret.web.Parse.Title.Regexp.Pattern != "" {
		_, err := regexp.Compile(ret.web.Parse.Title.Regexp.Pattern)
		if err != nil {
			return nil, fmt.Errorf("Parse.Title.Regexp.Patternを見直して下さい %w", err)
		}
	}
	if ret.web.Parse.Publisher.Regexp.Pattern != "" {
		_, err := regexp.Compile(ret.web.Parse.Publisher.Regexp.Pattern)
		if err != nil {
			return nil, fmt.Errorf("Parse.Publisher.Regexp.Patternを見直して下さい %w", err)
		}
	}
	if ret.web.Parse.Pubdate.Regexp.Pattern != "" {
		_, err := regexp.Compile(ret.web.Parse.Pubdate.Regexp.Pattern)
		if err != nil {
			return nil, fmt.Errorf("Parse.Pubdate.Regexp.Patternを見直して下さい %w", err)
		}
	}
	return ret, nil
}

func (bd *webSite) Get(isbn string) error {
	url := strings.Replace(bd.web.URL, "{{.ISBN}}", isbn, -1)
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if bd.web.UserAgent != "" {
		req.Header.Set("User-Agent", bd.web.UserAgent)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	bd.data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errors.New("Response: " + resp.Status)
	}
	return bd.parse()
}
func (bd *webSite) Save(path string) error {
	if bd.web.File == "" {
		return errors.New("YamlファイルにFileが設定されていません。")
	}
	json := filepath.Join(path, bd.web.File)
	return ioutil.WriteFile(json, bd.data, 0644)
}
func (bd *webSite) Load(path string) (err error) {
	if bd.web.File == "" {
		return errors.New("YamlファイルにFileが設定されていません。")
	}
	json := filepath.Join(path, bd.web.File)
	bd.data, err = ioutil.ReadFile(json)
	if err != nil {
		return
	}
	err = bd.parse()
	return
}

func (bd *webSite) parse() error {
	doc, err := htmlquery.Parse(bytes.NewReader(bd.data))
	if err != nil {
		return err
	}

	var author []string
	for _, item := range bd.web.Parse.Author.XPath {
		list := htmlquery.Find(doc, item)
		for _, tmp := range list {
			str := strings.TrimSpace(htmlquery.InnerText(tmp))
			str = bd.web.Parse.Author.Regexp.Replace(str)
			author = append(author, str)
		}
	}
	bd.Author = strings.Join(author, bd.web.Parse.Author.Join)

	var title []string
	for _, item := range bd.web.Parse.Title.XPath {
		list := htmlquery.Find(doc, item)
		for _, tmp := range list {
			str := strings.TrimSpace(htmlquery.InnerText(tmp))
			str = bd.web.Parse.Title.Regexp.Replace(str)
			title = append(title, str)
		}
	}
	bd.Title = strings.Join(title, bd.web.Parse.Title.Join)

	for _, item := range bd.web.Parse.Publisher.XPath {
		list := htmlquery.Find(doc, item)
		for _, tmp := range list {
			str := strings.TrimSpace(htmlquery.InnerText(tmp))
			str = bd.web.Parse.Publisher.Regexp.Replace(str)
			if str != "" {
				bd.Publisher = str
				break
			}
		}
		if bd.Publisher != "" {
			break
		}
	}
	for _, item := range bd.web.Parse.Pubdate.XPath {
		list := htmlquery.Find(doc, item)
		for _, tmp := range list {
			str := strings.TrimSpace(htmlquery.InnerText(tmp))
			str = bd.web.Parse.Pubdate.Regexp.Replace(str)
			if str != "" {
				bd.Pubdate = str
				break
			}
		}
		if bd.Pubdate != "" {
			break
		}
	}
	var number = regexp.MustCompile(`[\d\-]{9,}[xX]?`)
	var numberonly = regexp.MustCompile(`-`)
	for _, item := range bd.web.Parse.ISBN.XPath {
		list := htmlquery.Find(doc, item)
		for _, tmp := range list {
			str := htmlquery.InnerText(tmp)
			for _, n := range number.FindAllString(str, -1) {
				n = numberonly.ReplaceAllString(n, "")
				if len(n) == 10 {
					bd.ISBN = n
					break
				}
				if len(n) == 13 && (n[12] != 'x' && n[12] != 'X') {
					bd.ISBN = n
					break
				}
			}
			if bd.ISBN != "" {
				break
			}
		}
		if bd.ISBN != "" {
			break
		}
	}
	if bd.Author == "" {
		if bd.Publisher != "" {
			bd.Author = bd.Publisher
		} else {
			//authorが空白ならエラー
			return errors.New(bd.file + " Author not found")
		}
	}
	if bd.Title == "" {
		//titleが空白ならエラー
		return errors.New(bd.file + " Title not found")
	}
	return nil
}
