package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/antchfx/htmlquery"
	"github.com/axgle/mahonia"
	"golang.org/x/net/html"
	"golang.org/x/text/encoding/simplifiedchinese"
)

const (
	piliEndpoint = "https://www.pilibook.com"
	catgory      = "/7_"
	// novelsBatch  = 10
	pagesBatch = 2
)

var enc = simplifiedchinese.GBK

func main() {
	var endPage int
	allPages := 50
	for startPage := 1; startPage <= allPages; {
		endPage = startPage + pagesBatch
		wgGroup := &sync.WaitGroup{}
		wgGroup.Add(pagesBatch)
		for i := startPage; i < endPage; i++ {
			go func(indexPage int, wgGroup *sync.WaitGroup) {
				parseIndex(indexPage)
				wgGroup.Done()
			}(i, wgGroup)
		}
		wgGroup.Wait()
		startPage += endPage
	}
}

func parseIndex(indexPage int) {
	indexUrl := piliEndpoint + catgory + strconv.Itoa(indexPage) + "/"
	doc := fetch(indexUrl)
	nodes := htmlquery.Find(doc, `//div[@class="books"]/div/h3/a`)

	wgGroup := &sync.WaitGroup{}
	wgGroup.Add(len(nodes))

	for _, node := range nodes {
		go func(url string, wgGroup *sync.WaitGroup) {
			parseBookChaps(url)
			wgGroup.Done()
		}(htmlquery.SelectAttr(node, "href"), wgGroup)
	}
	wgGroup.Wait()
}

func parseBookChaps(url string) {
	doc := fetch(url)
	bookName := htmlquery.InnerText(htmlquery.FindOne(doc, `//div[@class="book_info"]/h1/text()`))
	bookName = cvtStrEncoding(bookName, "gbk", "utf-8")
	bookName += ".txt"

	file, err := os.Create("books/" + bookName)
	if err != nil {
		log.Fatal("file create err:", bookName, err)
	}

	nodes := htmlquery.Find(doc, `//div[@class="book_list"]/ul/li/a`)
	for _, node := range nodes {
		contentName := htmlquery.InnerText(node)
		contentName = cvtStrEncoding(contentName, "gbk", "utf-8")
		file.WriteString(contentName + "\n")
		content := parseChapContents(piliEndpoint + htmlquery.SelectAttr(node, "href"))
		file.WriteString(content + "\n")
	}
	if err = file.Close(); err != nil {
		log.Fatal(err)
	}
}

func parseChapContents(url string) (res string) {
	doc := fetch(url)
	nodes := htmlquery.Find(doc, `//div[@id="htmlContent"]/text()`)
	for _, node := range nodes {
		content := htmlquery.InnerText(node)
		content = cvtStrEncoding(content, "gbk", "utf-8")
		res += content
	}
	res = strings.ReplaceAll(res, "聽聽聽聽", "	")
	res = res[8:]
	return
}

func fetch(url string) *html.Node {
	log.Println("Fetch Url", url)
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Http get err:", err)
	}
	if resp.StatusCode != 200 {
		log.Fatal("Http status code:", resp.StatusCode)
	}
	defer resp.Body.Close()
	doc, err := htmlquery.Parse(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return doc
}

func cvtStrEncoding(src string, srcCode string, tagCode string) string {
	srcCoder := mahonia.NewDecoder(srcCode)
	srcResult := srcCoder.ConvertString(src)
	tagCoder := mahonia.NewDecoder(tagCode)
	_, cdata, _ := tagCoder.Translate([]byte(srcResult), true)
	return string(cdata)
}
