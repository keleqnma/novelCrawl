package getter

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/Aiicy/htmlquery"
	"github.com/go-clog/clog"
	"github.com/henson/proxypool/pkg/models"
)

func Xilidali() (result []*models.IP) {
	Threads := make(chan int, 5) //控制在5以内
	var wg sync.WaitGroup
	for i := 1; i < 2000; i++ {
		Threads <- 1
		wg.Add(1)
		go func(i int) {
			url := fmt.Sprintf("http://www.xiladaili.com/http/%d/", i)
			res, err := http.Get(url)
			if err != nil {
				log.Println(err)
				return
			}
			if res.Body == nil {
				return
			}
			if res.Body != nil {
				defer res.Body.Close()
			}
			root, _ := htmlquery.Parse(res.Body)
			tr, _ := htmlquery.Find(root, "//tr")
			for _, row := range tr {
				item, _ := htmlquery.Find(row, "//td")
				if len(item) > 0 {
					ip := htmlquery.InnerText(item[0])
					result = append(result, &models.IP{
						Data:  ip,
						Type1: "http",
					})
				}
			}
			defer wg.Done()
			<-Threads
		}(i)
	}
	wg.Wait()
	clog.Info("[Xilidali] done")
	return result
}
