package main

import (
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/antchfx/htmlquery"
	"github.com/axgle/mahonia"
	"golang.org/x/net/html"
)

const (
	piliEndpoint  = "https://www.pilibook.com"
	danmeiCatgory = "/7_"
	novelPath     = "testMultiReqs/"
	pagesBatch    = 5
	allPages      = 50
	beginPage     = 1
	maxRetryTimes = 10
)

var novelNums int32

func main() {
	var endPage int
	beginTime := time.Now()
	novelNums = 0
	startPage := beginPage
	log.Printf("开始爬取, 当前时间:%v\n", beginTime)

	for startPage <= allPages {
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
		startPage = endPage
	}

	endTime := time.Now()
	log.Printf("全部爬取完毕, 当前时间:%v, 耗时：%v, 共爬取 %v 本小说\n", endTime, endTime.Sub(beginTime), novelNums)
}

func parseIndex(indexPage int) {
	indexUrl := piliEndpoint + danmeiCatgory + strconv.Itoa(indexPage) + "/"
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
	log.Printf("page %v 爬取完毕\n", indexPage)
	log.Print("--------------------------------------------")
}

func parseBookChaps(url string) {
	doc := fetch(url)
	bookName := htmlquery.InnerText(htmlquery.FindOne(doc, `//div[@class="book_info"]/h1/text()`))
	bookName = cvtStrEncoding(bookName, "gbk", "utf-8")
	bookName += ".txt"

	file, err := os.Create(novelPath + bookName)
	if err != nil {
		log.Fatal("file create err:", bookName, err)
	}

	nodes := htmlquery.Find(doc, `//div[@class="book_list"]/ul/li/a`)
	for _, node := range nodes {
		contentName := htmlquery.InnerText(node)
		contentName = cvtStrEncoding(contentName, "gbk", "utf-8")
		file.WriteString(contentName + "\n")
		content := parseChapContents(piliEndpoint + htmlquery.SelectAttr(node, "href"))
		file.WriteString(content)
	}
	if err = file.Close(); err != nil {
		log.Fatal(err)
	}
	atomic.AddInt32(&novelNums, 1)
	log.Printf("小说 %v 爬取完毕\n", bookName)
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
	var (
		resp       *http.Response
		retryTimes float64
		err        error
	)

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")

	for resp, err = client.Do(req); (err != nil || resp.StatusCode != 200) && retryTimes < maxRetryTimes; {
		log.Printf("Http get err:", err)
		log.Printf("Http status code:", resp.StatusCode)
		log.Printf("Retry %v times", retryTimes)
		time.Sleep(time.Microsecond * time.Duration((math.Pow(2, retryTimes))))
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
