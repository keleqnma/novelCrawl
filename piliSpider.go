package cocoSpider

import (
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/antchfx/htmlquery"
)

type CrawlNode struct {
	baseURL string
	batch   int

	nameRule    ParseRule
	contextRule ParseRule
	childRule   ParseRule

	childNodes []*CrawlNode
}

func (cNode *CrawlNode) Crawl() {
	beginTime := time.Now()
	log.Printf("开始爬取节点%v, 当前时间:%v\n", cNode.baseURL, beginTime)

	doc := fetch(cNode.baseURL)
	cNode.nameRule.parse(doc)
	cNode.contextRule.parse(doc)
	cNode.childRule.parse(doc)

	finCh := make(chan (int), 1)
	finCh <- cNode.batch
	wg := &sync.WaitGroup{}
	for index := 0; index < len(cNode.childNodes); {
		pages := <-finCh
		wg.Add(pages)
		for i := 0; i < pages; i++ {
			go func(index int, finCh chan int, wg *sync.WaitGroup) {
				cNode.childNodes[index].Crawl()
				finCh <- 1
				wg.Done()
			}(index, finCh, wg)
			index++
		}
	}
	wg.Wait()
	endTime := time.Now()
	log.Printf("%v爬取完毕, 当前时间:%v, 耗时：%v,\n", cNode.baseURL, endTime, endTime.Sub(beginTime))
}

const (
	piliEndpoint  = "http://www.yuzhaiwu520.org"
	danmeiCatgory = "/danmei/7_"
	novelPath     = "download/"
	pagesBatch    = 5
	beginPage     = 1
	endPage       = 296
	maxRetryTimes = 10
)

var novelNums int32

func Crawl() {
	beginTime := time.Now()
	novelNums = 0
	log.Printf("开始爬取, 当前时间:%v\n", beginTime)

	finCh := make(chan (int), 1)
	finCh <- 1
	wg := &sync.WaitGroup{}
	wg.Add(endPage - beginPage + 1)

	for indexPage := beginPage; indexPage < endPage; {
		pages := <-finCh
		for i := 0; i < pages; i++ {
			go func(indexPage int, finCh chan int, wg *sync.WaitGroup) {
				crawlPages(indexPage)
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

func crawlPages(indexPage int) {
	indexUrl := piliEndpoint + danmeiCatgory + strconv.Itoa(indexPage) + ".html"
	log.Printf("开始爬取 page %v\n", indexUrl)
	log.Print("--------------------------------------------")

	doc := fetch(indexUrl)
	nodes := htmlquery.Find(doc, `//div[@class="l"]/ul/li/span[1]/a`)

	finCh := make(chan (int), 1)
	finCh <- pagesBatch
	wg := &sync.WaitGroup{}
	wg.Add(len(nodes))

	for cur := 0; cur < len(nodes); {
		pages := <-finCh
		for i := 0; i < pages; i++ {
			go func(cur int, finCh chan int, wg *sync.WaitGroup) {
				crawlBookChaps(htmlquery.SelectAttr(nodes[cur], "href"))
				finCh <- 1
				wg.Done()
			}(cur, finCh, wg)
			cur++
		}
	}

	// for _, node := range nodes {
	// 	go func(url string, wgGroup *sync.WaitGroup) {
	// 		crawlBookChaps(url)
	// 		wgGroup.Done()
	// 	}(htmlquery.SelectAttr(node, "href"), wg)
	// }

	wg.Wait()
	log.Printf("page %v 爬取完毕\n", indexPage)
	log.Print("--------------------------------------------")
}

func crawlBookChaps(url string) {
	doc := fetch(url)
	bookName := htmlquery.InnerText(htmlquery.FindOne(doc, `//div[@id="info"]/h1/text()`))
	bookName = cvtStrEncoding(bookName, "gbk", "utf-8")
	bookName += ".txt"
	bookName = strings.ReplaceAll(bookName, "/", "or")

	file, err := os.Create(novelPath + bookName)
	if err != nil {
		log.Fatal("file create err: ", bookName, err)
	}
	log.Printf("开始爬取小说 %v \n", bookName)
	nodes := htmlquery.Find(doc, `//dl/dd/a`)
	for _, node := range nodes {
		contentName := htmlquery.InnerText(node)
		contentName = cvtStrEncoding(contentName, "gbk", "utf-8")
		if _, err = file.WriteString(contentName + "\n"); err != nil {
			log.Fatal(err)
		}
		content := crawlChapContents(piliEndpoint + htmlquery.SelectAttr(node, "href"))
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

func crawlChapContents(url string) (res string) {
	doc := fetch(url)
	nodes := htmlquery.Find(doc, `//div[@id='content']/text()`)
	for _, node := range nodes {
		content := htmlquery.InnerText(node)
		content = cvtStrEncoding(content, "gbk", "utf-8")
		res += content
	}
	res = strings.ReplaceAll(res, "聽聽聽聽", "	")
	res = res[8:]
	return
}
