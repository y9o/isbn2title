package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"

	_ "golang.org/x/image/bmp"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/oned"
)

type option struct {
	row        int
	input      string
	headCount  int
	tailCount  int
	noRotate   bool
	noAccess   bool
	noRename   bool
	save       bool
	test       bool
	rename     string
	ISBN       string
	API        string
	check      string
	checknames bool
}

var isbnScanner = oned.NewEAN13Reader()

func main() {
	var op option
	flag.IntVar(&op.row, "row", 100, "画像を上下に分割してスキャン")
	flag.IntVar(&op.headCount, "head", 5, "見つかるまでスキャンするファイルの数")
	flag.IntVar(&op.tailCount, "tail", 5, "フォルダ内の最後尾から見つかるまでスキャンするファイルの数")
	flag.BoolVar(&op.noRotate, "noRotate", false, "横向き画像を想定した、回転して再スキャンをしない")
	flag.BoolVar(&op.noAccess, "noAccess", false, "ISBN取得後、WebAPIに接続せず終了する")
	flag.BoolVar(&op.noRename, "noRename", false, "WebAPIから取得後、フォルダ名を変更しない")
	flag.BoolVar(&op.save, "save", false, "WebAPIから取得したデータをファイルに保存する")
	flag.BoolVar(&op.test, "test", false, "保存されたデータを読み込んで-renameをテスト")
	flag.StringVar(&op.rename, "rename", "[{{.Author}}] {{.Title}} {{with .Publisher}}[{{.}}]{{end}}{{with .Pubdate}}[{{.}}]{{end}}[ISBN {{.ISBN}}]", "新しいフォルダ名")
	flag.StringVar(&op.API, "API", "openbd,google,kokkai", "使用するWebAPIとアクセス順番")
	flag.StringVar(&op.check, "check", "", "ISBN13が記入されたファイルのパス。存在すればバーコードスキャンをしない")
	flag.BoolVar(&op.checknames, "checknames", false, "フォルダ名からISBN番号を読み取る")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] フォルダパス \n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "　指定されたフォルダ内の画像(jpg,bmp,png)からISBNバーコードをスキャンして、WebAPIから取得した情報でフォルダ名を変更する。\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if op.row < 1 {
		op.row = 100
	}

	apis := make([]isbnAPI, 0, 3)
	for _, apiname := range strings.Split(op.API, ",") {
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
				log.Fatalf("(%s) %s\n", apiname, err)
			} else {
				apis = append(apis, api)
			}
		}
	}

	if op.test {
		log.Printf("\"%s\" をテストします。\n", op.rename)
		for _, api := range apis {
			if err := api.Load(op.input); err != nil {
				if os.IsNotExist(err) {
					log.Printf("test(%T): データありません(%s)\n", api, err)
				} else {
					log.Printf("test(%T): %s\n", api, err)
				}
			} else {
				newname := makeFileNameFromBD(api, &op)
				log.Printf("test(%T): => \"%s\"\n", api, newname)
			}
		}
		return
	}

	op.input = flag.Arg(0)
	if op.input == "" {
		log.Fatalln("対象フォルダを指定してください(-help)")
		return
	}

	stat, err := os.Stat(op.input)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatalln("存在しないフォルダです")
		}
		log.Fatalln(err)
	} else if stat.IsDir() {
		if op.check != "" {
			isbn13, err := ioutil.ReadFile(op.check)
			if err == nil {
				tmp := regexp.MustCompile(`\D`).ReplaceAll(isbn13, nil)
				if len(tmp) == 13 {
					op.ISBN = string(tmp)
					log.Printf("check: %s\n", op.check)
				}
			}
		} else if op.checknames {
			r := regexp.MustCompile(`\d{13}|\d{9}[xX]|\d{10}`)
			list := r.FindAllString(op.input, -1)
			for _, txt := range list {
				op.ISBN = txt
				break
			}
		}
		if op.ISBN == "" {
			isbn := checkDir(&op)
			if isbn != "" {
				op.ISBN = isbn
			}
		}
	} else {
		log.Fatalln("対象フォルダを指定してください")
	}

	if op.ISBN != "" {
		log.Printf("ISBN: %s\n", op.ISBN)
	} else {
		log.Fatalln("バーコードが見つかりませんでした")
	}

	if op.noAccess {
		return
	}

	for _, api := range apis {
		err := api.Get(op.ISBN)
		if err != nil {
			log.Println(err)
			continue
		}
		if op.save {
			err := api.Save(op.input)
			if err != nil {
				log.Println(err)
			}
		}
		newname := makeFileNameFromBD(api, &op)
		if newname != "" {
			olddir, oldname := filepath.Split(filepath.Clean(op.input))
			log.Printf("rename: %s => %s\n", oldname, newname)
			if oldname != newname {
				newpath := filepath.Join(olddir, newname)
				if !op.noRename {
					if _, err := os.Stat(newpath); os.IsNotExist(err) {
						err = os.Rename(op.input, newpath)
						if err != nil {
							log.Println(err)
						}
					} else {
						log.Println("すでに同名のファイルが存在します")
					}
				}
			}
		}
		break
	}
}

//フォルダ内からファイルリストを作成
func checkDir(op *option) string {
	files, err := ioutil.ReadDir(op.input)
	if err != nil {
		log.Fatalln(err)
	}
	if op.headCount > 0 {
		txt := checkFiles(files, op.headCount, op)
		if txt != "" {
			return txt
		}
	}
	if op.tailCount > 0 {
		//reverse files
		for i := len(files)/2 - 1; i >= 0; i-- {
			opp := len(files) - 1 - i
			files[i], files[opp] = files[opp], files[i]
		}
		return checkFiles(files, op.tailCount, op)
	}
	return ""
}

//ファイルリストから画像を探してスキャン
func checkFiles(files []os.FileInfo, count int, op *option) string {
	for _, file := range files {
		if count <= 0 {
			break
		}
		if file.IsDir() {
			continue
		}

		fh, err := os.Open(filepath.Join(op.input, file.Name()))
		if err != nil {
			log.Println(err)
			continue
		}
		defer fh.Close()
		img, _, err := image.Decode(fh)
		if err != nil {
			continue
		}
		log.Printf("Scan: %s\n", file.Name())
		isbn := getISBNfromImage(img, op)
		if isbn != "" {
			return isbn
		}
		count--
	}
	return ""
}

//画僧をスキャンして、見つからなければ回転してもう一度スキャン
func getISBNfromImage(img image.Image, op *option) string {
	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		log.Fatalln(err)
	}

	if !bmp.IsCropSupported() {
		result, err := isbnScanner.Decode(bmp, nil)
		if err == nil {
			return result.GetText()
		}
		log.Fatalln("!bmp.IsCropSupported()")
	}

	txt := getISBNfromBmp(bmp, op)
	if txt != "" {
		return txt
	}
	if !op.noRotate {
		if !bmp.IsRotateSupported() {
			log.Fatalln("!bmp.IsRotateSupported()")
		}
		bmp90, err := bmp.RotateCounterClockwise()
		if err != nil {
			log.Fatalln(err)
		}
		txt = getISBNfromBmp(bmp90, op)
	}
	return txt
}

//画像をスキャン バーコードが2つあるので、画像を細かく区切って上から検索する必要がある
func getISBNfromBmp(bmp *gozxing.BinaryBitmap, op *option) string {
	height := bmp.GetHeight()
	width := bmp.GetWidth()
	cropHeight := height / op.row
	if cropHeight < 1 {
		cropHeight = 1
	}

	for top := 0; top+cropHeight <= height; top += cropHeight {
		//分割
		newBmp, err := bmp.Crop(0, top, width, cropHeight)
		if err != nil {
			log.Fatalf("!bmp.Crop(%d,%d,%d,%d)\n", 0, top, width, cropHeight)
		}
		//バーコードを探す
		result, err := isbnScanner.DecodeWithoutHints(newBmp)
		if err == nil {
			txt := result.GetText()
			//ISBNはいずれかで始まる
			if strings.HasPrefix(txt, "978") || strings.HasPrefix(txt, "979") {
				return txt
			}
		}
	}

	return ""
}

//WebAPIのデータからファイル名を作成
func makeFileNameFromBD(data isbnAPI, op *option) string {

	tmpl, err := template.New("name").Funcs(template.FuncMap{"hasField": hasField}).Parse(op.rename)
	if err != nil {
		log.Println(err)
		return ""
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		log.Println(err)
		return strings.TrimSpace(replaceFileName(buf.String()))
	}
	return strings.TrimSpace(replaceFileName(buf.String()))
}

//ファイル名 禁止文字列
func replaceFileName(old string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r <= 0x1f:
			return ' '
		}
		switch r {
		case 0x7f:
			return ' '
		case '"':
			return '”'
		case '*':
			return '＊'
		case '/':
			return '／'
		case ':':
			return '：'
		case '<':
			return '＜'
		case '>':
			return '＞'
		case '?':
			return '？'
		case '\\':
			return '￥'
		case '|':
			return '｜'
		}
		return r
	}, old)
}

//https://stackoverflow.com/questions/34703133/field-detection-in-go-html-template
func hasField(v interface{}, name string) bool {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return false
	}
	return rv.FieldByName(name).IsValid()
}
