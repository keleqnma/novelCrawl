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
	endPage       = 50
	beginPage     = 1
	maxRetryTimes = 10
)

var novelNums int32

func main() {
	beginTime := time.Now()
	novelNums = 0
	log.Printf("开始爬取, 当前时间:%v\n", beginTime)

	finCh := make(chan (int), 1)
	finCh <- pagesBatch
	wg := &sync.WaitGroup{}

	for indexPage := beginPage; indexPage < endPage; {
		pages := <-finCh
		wg.Add(pages)
		for i := 0; i < pages; i++ {
			go func(indexPage int, finCh chan int, wg *sync.WaitGroup) {
				parseIndex(indexPage)
				finCh <- 1
				wg.Done()
			}(indexPage, finCh, wg)
			indexPage++
		}
	}

	wg.Wait()
	endTime := time.Now()
	log.Printf("全部爬取完毕, 当前时间:%v, 耗时：%v, 共爬取 %v 本小说\n", endTime, endTime.Sub(beginTime), novelNums)
}

func parseIndex(indexPage int) {
	log.Printf("开始爬取 page %v\n", indexPage)
	log.Print("--------------------------------------------")

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
		if _, err = file.WriteString(contentName + "\n"); err != nil {
			log.Fatal(err)
		}
		content := parseChapContents(piliEndpoint + htmlquery.SelectAttr(node, "href"))
		if _, err = file.WriteString(content); err != nil {
			log.Fatal(err)
		}
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
