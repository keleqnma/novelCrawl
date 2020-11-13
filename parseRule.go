package cocoSpider

import "golang.org/x/net/html"

type Processer interface {
	Process(str *string)
}

type ParseRule struct {
	ParsePattern string
	Processer    Processer
}

func (rule *ParseRule) parse(doc *html.Node) {

}
