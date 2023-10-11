package main

import (
	"jktimeScraper/pkg/config"
	"jktimeScraper/pkg/log"
	"testing"
)

func TestDownload(t *testing.T) {
	conf := config.NewConfig()
	logger = log.NewLog(conf)
	//  100312001/631938
	downloadArticle(100312001, 631938)
}

func TestHtml2md(t *testing.T) {
	conf := config.NewConfig()
	logger = log.NewLog(conf)

	html2md("./articles/100081501/385003/article_content_385003")
}
