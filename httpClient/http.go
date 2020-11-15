package httpClient

import (
	"cocoSpider/dnsCache"
	"cocoSpider/httpClient/proxyPool"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

type Client struct {
	dnsCache   *dnsCache.Resolver
	proxyPool  *proxyPool.ProxyPool
	clients    []*http.Client
	clientSize int
}

const maxRetryTimes = 12

func New(clientSize int, endPoint string) *Client {
	client := &Client{}
	client.dnsCache = dnsCache.New(time.Minute * 5)
	client.proxyPool = proxyPool.New(endPoint)
	for i := 0; i < clientSize; i++ {
		proxy, err := url.Parse(client.proxyPool.GetProxyIP())
		if err != nil {
			log.Printf("parse proxy failed:%v\n", err)
			continue
		}
		client.clients = append(client.clients, &http.Client{
			Transport: &http.Transport{
				Proxy:               http.ProxyURL(proxy),
				MaxIdleConnsPerHost: clientSize * 2,
				Dial: func(network string, address string) (net.Conn, error) {
					separator := strings.LastIndex(address, ":")
					ip, _ := client.dnsCache.FetchOneString(address[:separator])
					return net.Dial("tcp", ip+address[separator:])
				},
			},
		})
	}
	client.clientSize = clientSize
	return client
}

func (client *Client) getClient() *http.Client {
	return client.clients[rand.Int()%client.clientSize]
}

func (client *Client) Fetch(endPoint string) *html.Node {
	var (
		resp       *http.Response
		retryTimes float64
		err        error
		doc        *html.Node
	)

	req, _ := http.NewRequest("GET", endPoint, nil)
	for retryTimes < maxRetryTimes {
		req.Header.Set("User-Agent", getAgent())
		c := client.getClient()
		resp, err = c.Do(req)
		if err == nil && (resp != nil && resp.StatusCode == 200) {
			break
		}
		log.Printf("Http get err: %v, url: %v", err, endPoint)
		if resp != nil {
			log.Printf("Http Responce: %v", resp.StatusCode)
		}
		log.Printf("Retry %v times", retryTimes)
		time.Sleep(time.Second * time.Duration((math.Pow(5, retryTimes))))
		retryTimes++
	}

	if resp != nil {
		defer resp.Body.Close()
		doc, err = htmlquery.Parse(resp.Body)
		if err != nil {
			log.Println(err)
		}
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
