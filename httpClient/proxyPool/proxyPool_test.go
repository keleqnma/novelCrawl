package proxyPool

import "testing"

func TestGetProxyIP(t *testing.T) {
	proxyPool := New("http://www.yuzhaiwu520.org")
	proxyPool.GetProxyIP()
}
