package proxyPool

import (
	"cocoSpider/httpClient/proxyPool/getter"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/henson/proxypool/pkg/models"
)

type ProxyPool struct {
	mu       *sync.RWMutex
	ips      []*models.IP
	funcs    []func() []*models.IP
	endPoint string
	total    int
}

func New(endPoint string) *ProxyPool {
	p := &ProxyPool{}
	p.mu = &sync.RWMutex{}
	p.funcs = []func() []*models.IP{
		getter.IP3306,
		getter.KDL,
		getter.IP89,
		getter.Xilidali,
	}
	p.endPoint = endPoint
	p.fetchOnceProxys()
	go p.FetchProxys()
	return p
}

func (p *ProxyPool) fetchOnceProxys() {
	var wg sync.WaitGroup
	for _, f := range p.funcs {
		wg.Add(1)
		go func(f func() []*models.IP) {
			ips := f()
			wg.Add(len(ips))
			for _, ip := range ips {
				go func(ip *models.IP) {
					if speed, status := proxyTest(processIP(ip), p.endPoint); status == 200 && speed <= 2000 {
						p.mu.Lock()
						p.total++
						p.ips = append(p.ips, ip)
						p.mu.Unlock()
					}
					wg.Done()
				}(ip)
			}
			wg.Done()
		}(f)
	}
	wg.Wait()
	if p.total == 0 {
		log.Fatal("ip proxy pool was broken.")
	}
	log.Printf("All getters finished,ips length:%v.\n", p.total)
}

func (p *ProxyPool) FetchProxys() {
	for {
		time.Sleep(time.Minute * 10)
		p.fetchOnceProxys()
	}
}

func (p *ProxyPool) GetProxy() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ip := p.ips[rand.Int()%p.total]
	return processIP(ip)
}

func processIP(ip *models.IP) string {
	if ip.Type1 == "https" {
		return "https://" + ip.Data
	}
	return "http://" + ip.Data
}

func proxyTest(proxyIP string, endPoint string) (Speed int, Status int) {
	// 解析代理地址
	proxy, err := url.Parse(proxyIP)
	if err != nil {
		log.Printf("parse proxy failed:%v\n", err)
		return
	}
	//设置网络传输
	netTransport := &http.Transport{
		Proxy:                 http.ProxyURL(proxy),
		MaxIdleConnsPerHost:   10,
		ResponseHeaderTimeout: time.Second * time.Duration(50),
	}
	// 创建连接客户端
	httpClient := &http.Client{
		Timeout:   time.Second * 10,
		Transport: netTransport,
	}
	begin := time.Now() //判断代理访问时间
	// 使用代理IP访问测试地址
	res, err := httpClient.Get(endPoint)
	if err != nil {
		log.Println(err)
		return
	}
	defer res.Body.Close()
	speed := int(time.Since(begin).Nanoseconds() / 1000 / 1000) //ms
	//判断是否成功访问，如果成功访问StatusCode应该为200
	if res.StatusCode != http.StatusOK {
		log.Println(err)
		return
	}
	return speed, res.StatusCode
}
