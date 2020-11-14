package httpClient

import (
	"cocoSpider/dnsCache"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

type Client struct {
	c        *http.Client
	dnsCache *dnsCache.Resolver
}

const maxRetryTimes = 10

func New() *Client {
	client := &Client{}
	client.dnsCache = dnsCache.New(time.Minute * 5)
	client.c = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 64,
			Dial: func(network string, address string) (net.Conn, error) {
				separator := strings.LastIndex(address, ":")
				ip, _ := client.dnsCache.FetchOneString(address[:separator])
				return net.Dial("tcp", ip+address[separator:])
			},
		},
	}
	return client
}

func (client *Client) Fetch(url string) *html.Node {
	var (
		resp       *http.Response
		retryTimes float64
		err        error
	)

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Set("User-Agent", getAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Connection", "keep-alive")

	for resp, err = client.c.Do(req); (err != nil || (resp != nil && resp.StatusCode != 200)) && retryTimes < maxRetryTimes; retryTimes += 1 {
		log.Printf("Http get err: %v, url: %v, req: %v", err, url, req)
		if resp != nil {
			log.Printf("Http Responce: %v", resp)
		}
		log.Printf("Retry %v times", retryTimes)
		time.Sleep(time.Second * time.Duration((math.Pow(2, retryTimes))))
		req.Header.Set("User-Agent", getAgent())
	}

	defer resp.Body.Close()
	doc, err := htmlquery.Parse(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return doc
}

func getAgent() string {
	agent := [...]string{
		"Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:50.0) Gecko/20100101 Firefox/50.0",
		"Opera/9.80 (Macintosh; Intel Mac OS X 10.6.8; U; en) Presto/2.8.131 Version/11.11",
		"Opera/9.80 (Windows NT 6.1; U; en) Presto/2.8.131 Version/11.11",
		"Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 5.1; 360SE)",
		"Mozilla/5.0 (Windows NT 6.1; rv:2.0.1) Gecko/20100101 Firefox/4.0.1",
		"Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 5.1; The World)",
		"User-Agent,Mozilla/5.0 (Macintosh; U; Intel Mac OS X 10_6_8; en-us) AppleWebKit/534.50 (KHTML, like Gecko) Version/5.1 Safari/534.50",
		"User-Agent, Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 5.1; Maxthon 2.0)",
		"User-Agent,Mozilla/5.0 (Windows; U; Windows NT 6.1; en-us) AppleWebKit/534.50 (KHTML, like Gecko) Version/5.1 Safari/534.50",
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	len := len(agent)
	return agent[r.Intn(len)]
}
