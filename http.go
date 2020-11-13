package cocoSpider

import (
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

func fetch(url string) *html.Node {
	var (
		resp       *http.Response
		retryTimes float64
		err        error
	)

	client := &http.Client{
		Transport: &http.Transport{
			//TODO
		},
	}
	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Set("User-Agent", getAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Connection", "keep-alive")

	for resp, err = client.Do(req); (err != nil || (resp != nil && resp.StatusCode != 200)) && retryTimes < maxRetryTimes; {
		log.Printf("Http get err: %v, url: %v", err, url)
		if resp != nil {
			log.Printf("Http status code: %v", resp.StatusCode)
		}
		log.Printf("Retry %v times", retryTimes)
		time.Sleep(time.Millisecond * time.Duration((math.Pow(10, retryTimes))))
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
