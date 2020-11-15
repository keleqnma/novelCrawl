package proxyPool

import (
	"cocoSpider/httpClient/proxyPool/getter"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/go-clog/clog"
	"github.com/henson/proxypool/pkg/models"
)

const (
	DefaultTimeOut    = time.Second * 5
	DefaultTestRounds = 5
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
	go p.fetchProxys()
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
					p.proxyTest(ip)
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

func (p *ProxyPool) fetchProxys() {
	for {
		time.Sleep(time.Minute * 10)
		p.fetchOnceProxys()
	}
}

func (p *ProxyPool) GetProxyIP() string {
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

func (p *ProxyPool) proxyTest(ip *models.IP) {
	// 解析代理地址
	proxy, err := url.Parse(processIP(ip))
	if err != nil {
		log.Printf("parse proxy failed:%v\n", err)
		return
	}
	//设置网络传输, 创建连接客户端
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyURL(proxy),
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: DefaultTimeOut,
		},
	}
	for i := 0; i < DefaultTestRounds; i++ {
		begin := time.Now()
		res, err := httpClient.Get(p.endPoint)
		if err != nil {
			return
		}
		defer res.Body.Close()
		timePassed := time.Since(begin)
		if res.StatusCode != http.StatusOK {
			return
		}
		if timePassed > DefaultTimeOut {
			return
		}
	}
	p.mu.Lock()
	p.total++
	p.ips = append(p.ips, ip)
	p.mu.Unlock()
	clog.Info("[proxy-test] %v test success.", ip.Data)
}
