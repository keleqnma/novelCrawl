package cocoSpider

import (
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cocoSpider/httpClient"

	"github.com/antchfx/htmlquery"
)

type CrawlNode struct {
	baseURL string
	batch   int

	nameRule    ParseRule
	contextRule ParseRule
	childRule   ParseRule

	childNodes []*CrawlNode
	client     *httpClient.Client
}

func (cNode *CrawlNode) Crawl() {
	beginTime := time.Now()
	log.Printf("开始爬取节点%v, 当前时间:%v\n", cNode.baseURL, beginTime)
	cNode.client = httpClient.New(cNode.batch, cNode.baseURL)
	doc := cNode.client.Fetch(cNode.baseURL)
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
	endpoint      = "http://www.yuzhaiwu520.org"
	catgory       = "/danmei/7_"
	novelPath     = "download/"
	booksBatch    = 60
	beginPage     = 1
	endPage       = 296
	sleepInterval = 10
)

var novelNums int32
var client *httpClient.Client

func Crawl() {
	beginTime := time.Now()
	client = httpClient.New(booksBatch, endpoint)
	novelNums = 0
	log.Printf("开始爬取, 当前时间:%v\n", beginTime)

	finCh := make(chan (int), booksBatch)
	for i := 1; i <= booksBatch/2; i++ {
		finCh <- 2
	}
	finCh <- booksBatch
	wg := &sync.WaitGroup{}

	for indexPage := beginPage; indexPage < endPage; indexPage++ {
		go func(indexPage int, finCh chan int, wg *sync.WaitGroup) {
			crawlPages(indexPage, finCh, wg)
		}(indexPage, finCh, wg)
		time.Sleep(time.Second * 2)
	}

	wg.Wait()
	endTime := time.Now()
	log.Printf("全部爬取完毕, 当前时间:%v, 耗时：%v, 共爬取 %v 本小说\n", endTime, endTime.Sub(beginTime), novelNums)
}

func crawlPages(indexPage int, finCh chan int, wg *sync.WaitGroup) {
	indexUrl := endpoint + catgory + strconv.Itoa(indexPage) + ".html"
	doc := client.Fetch(indexUrl)
	if doc == nil {
		log.Printf("page %v 解析失败\n", indexUrl)
		return
	}
	nodes := htmlquery.Find(doc, `//div[@class="l"]/ul/li/span[1]/a`)
	pageWg := &sync.WaitGroup{}
	pageWg.Add(len(nodes))
	for cur := 0; cur < len(nodes); {
		books := <-finCh
		for i := 0; i < books && cur < len(nodes); i++ {
			wg.Add(1)
			go func(cur int, finCh chan int, wg *sync.WaitGroup) {
				crawlBookChaps(htmlquery.SelectAttr(nodes[cur], "href"))
				finCh <- 1
				wg.Done()
			}(cur, finCh, wg)
			cur++
		}
	}
	pageWg.Wait()
}

func crawlBookChaps(url string) {
	doc := client.Fetch(url)
	if doc == nil {
		log.Printf("book %v 解析失败\n", url)
		return
	}
	bookName := htmlquery.InnerText(htmlquery.FindOne(doc, `//div[@id="info"]/h1/text()`))
	bookName = cvtStrEncoding(bookName, "gbk", "utf-8")
	bookName += ".txt"
	bookName = strings.ReplaceAll(bookName, "/", "or")

	file, err := os.Create(novelPath + bookName)
	if err != nil {
		log.Fatal("file create err: ", bookName, err)
	}
	log.Printf("开始爬取小说 %v \n", bookName)
	time.Sleep(time.Second * time.Duration(rand.Int()%sleepInterval))
	nodes := htmlquery.Find(doc, `//dl/dd/a`)
	for _, node := range nodes {
		contentName := htmlquery.InnerText(node)
		contentName = cvtStrEncoding(contentName, "gbk", "utf-8")
		if _, err = file.WriteString(contentName + "\n"); err != nil {
			log.Fatal(err)
		}
		content := crawlChapContents(endpoint + htmlquery.SelectAttr(node, "href"))
		if _, err = file.WriteString(content); err != nil {
			log.Fatal(err)
		}
	}
	if err = file.Close(); err != nil {
		log.Fatal(err)
	}
	atomic.AddInt32(&novelNums, 1)
	log.Printf("小说 %v 爬取完毕, 共爬取 %v 本小说\n", bookName, novelNums)
}

func crawlChapContents(url string) (res string) {
	doc := client.Fetch(url)
	if doc == nil {
		log.Printf("content %v 解析失败\n", url)
		return
	}
	nodes := htmlquery.Find(doc, `//div[@id='content']/text()`)
	for _, node := range nodes {
		content := htmlquery.InnerText(node)
		content = cvtStrEncoding(content, "gbk", "utf-8")
		res += content
	}
	res = strings.ReplaceAll(res, "聽", " ")
	if len(res) > 8 {
		res = res[8:]
	}
	return
}
