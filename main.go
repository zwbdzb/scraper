package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/gocolly/colly"
	"github.com/hashicorp/go-retryablehttp"
	jsoniter "github.com/json-iterator/go"

	"zwbdzb.github.com/scraper/pkg/log"

	"zwbdzb.github.com/scraper/pkg/config"

	md "github.com/JohannesKaufmann/html-to-markdown"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary
var logger *log.Logger

const host string = "http://study.dss.17usoft.com"

var retryclient *retryablehttp.Client

// var wg sync.WaitGroup

func init() {
	retryclient = retryablehttp.NewClient() // 确认是不是线程安全的
	retryclient.RetryMax = 3
	retryclient.Logger = nil
	retryclient.HTTPClient.Timeout = 100 * time.Second
}

func main() {
	conf := config.NewConfig()
	logger = log.NewLog(conf)

	time1 := time.Now()
	// ---  cources
	var uri = "/articles/columns_data.json"
	var dir = "./articles"
	body, err := downloadJson(uri, dir)
	if err != nil {
		logger.Error("course list error")
		panic("course list error")
	}
	var courses []course
	err = json.Unmarshal(body, &courses)
	if err != nil {
		logger.Error("course list error")
	}
	logger.Sugar().Infof("course count : %d", len(courses))

	// --- coursess
	for _, c := range courses { // 并发下载全部225门课程， 平均每个课程30 课程，在启动初期会有大量的http并发（9000+）占用大量文件描述符，后期会逐渐减少， 导致goroutine执行时间不可控
		uri = fmt.Sprintf("/articles/%d/article_list_%d.json", c.Column_sku, c.Column_sku)
		dir = fmt.Sprintf("./articles/%d", c.Column_sku)
		body, err = downloadJson(uri, dir)
		if err != nil {
			return
		}
		var articles []article
		err := json.Unmarshal(body, &articles)
		if err != nil {
			logger.Error("articles list error")
			return
		}
		logger.Sugar().Infof("course %d have  %d article", c.Column_sku, len(articles))

		var wg sync.WaitGroup
		// ---  article content json
		for _, a := range articles { // 并发爬取一个课程的所有文章资源，每文章平均耗时40s，去平均为每课程耗时， 225课程耗时2.5 小时。
			wg.Add(1)
			go func(c course, a article) { // 下载内容的json文件，渲染html &&  爬取资源
				defer wg.Done()
				downloadArticle(c.Column_sku, a.Article_id)
			}(c, a)
		}
		wg.Wait()
	}

	logger.Sugar().Infof("scrape cost %s", time.Since(time1))
}

func downloadArticle(cid, aid int32) {
	time0 := time.Now()
	var ts = fmt.Sprintf("./articles/%d/%d/article_content_%d.txt", cid, aid, aid)
	if _, err := os.Stat(fmt.Sprintf("./articles/%d/%d/", cid, aid)); err == nil { // 文件夹存在
		if _, err := os.Stat(ts); errors.Is(err, os.ErrNotExist) { // 文件不存在
			return
		}
	}

	uri := fmt.Sprintf("/articles/%d/%d/article_content_%d.json", cid, aid, aid)
	dir := fmt.Sprintf("./articles/%d/%d", cid, aid)
	body, err := downloadJson(uri, dir)
	if err != nil {
		logger.Sugar().Errorf("request article %s  error: %v", uri, err)
		return
	}
	var ac article_content
	err = json.Unmarshal(body, &ac)
	if err != nil {
		logger.Sugar().Errorf("article %s content  Unmarshal error: %v", uri, err)
		return
	}
	time1 := time.Since(time0)
	filepath := fmt.Sprintf("./articles/%d/%d/article_content_%d.html", cid, aid, aid)
	err = renderHtml(filepath, ac)
	time2 := time.Since(time0)
	if err != nil {
		logger.Sugar().Errorf("generate article html %s  error: %v", uri, err)
		return
	}

	var fp = fmt.Sprintf("./articles/%d/%d/article_content_%d", cid, aid, aid)
	err = html2md(fp)
	time3 := time.Since(time0)
	if err != nil {
		logger.Sugar().Errorf("generate article markdown %s error: %v", uri, err)
		return
	}

	failed := assertResource(filepath)
	if failed != nil {
		time4 := time.Since(time0)
		logger.Sugar().With("time1", time1).With("time2", time2).With("time3", time3).With("time4", time4).Errorf(ts)
		makeTimeStamp(ts, failed)
		return
	} else {
		os.Remove(ts)
	}
}

// http://study.dss.17usoft.com/articles/100525001/article_list_100525001.json
/*
 uri: /articles/columns_data.json  标准资源路径
 dir： 待保存路径
*/
func downloadJson(uri string, dir string) ([]byte, error) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		os.MkdirAll(dir, 0777)
	}
	logger.Sugar().Infof("doanload url %s start", uri)
	resp, err := retryclient.Get(host + uri)
	if err != nil {
		logger.Sugar().Errorf("request %s error : %v", uri, err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		logger.Sugar().Errorf("request %s status code is not 200", uri)
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Sugar().Errorf("url %s response error: %v", uri, err)
		return nil, err
	}

	err = os.WriteFile("."+uri, body, 0777)
	if err != nil {
		logger.Sugar().Errorf("Error writing file %s error : %v", uri, err)
		return nil, err
	}
	return body, nil
}

func renderHtml(fileptah string, ac interface{}) error {
	var tmplFile = "article_content.tmpl"
	tmpl, err := template.New(tmplFile).ParseFiles(tmplFile)
	if err != nil {
		logger.Sugar().Errorf("renderhtml %s, error : %v", fileptah, err)
		return err
	}

	f, err := os.Create(fileptah)
	if err != nil {
		logger.Sugar().Errorf("create file  %s error: %v", fileptah, err)
		return err
	}
	err = tmpl.Execute(f, ac)
	if err != nil {
		logger.Sugar().Errorf(" generate html %s error: %v", fileptah, err)
		return err
	}
	logger.Sugar().Infof("generate html %s success", fileptah)
	return nil
}

func assertResource(filepath string) (failed []string) {
	jobCh := make(chan string)
	doneCh := make(chan struct{})
	errCh := make(chan string)
	// var wg sync.WaitGroup // 有个问题，waitgroup能等待所有goroutine结束，但是不知道goroutine 是否如预期执行完毕，需要其他机制（信道）来保证。
	t := &http.Transport{}
	t.RegisterProtocol("file", http.NewFileTransport(http.Dir(".")))

	c := colly.NewCollector()
	c.WithTransport(t) // 本地文件，需要加上这个

	// Find and visit all links
	c.OnHTML("img,source", func(e *colly.HTMLElement) { // 多选择器, 回调函数没有返回值，不知道资源是否下载成功，需要其他机制来保证。
		var href = ""
		uri := e.Attr("src")
		if uri == "" {
			return
		}
		if strings.Index(uri, "http://") > 0 { // 外站链接忽略
			return
		}
		if uri[0] == '.' { // ./相对路径改造
			href = host + uri[1:]
		}
		// wg.Add(1)
		logger.Sugar().Debugf("Search href : %s", href)
		go func(href string, dir string, jc chan string, ec chan string, dc chan struct{}) {
			// defer wg.Done()
			jobCh <- href
			logger.Sugar().Debugf("download %s assert start", href)
			res, err := retryclient.Get(href)
			if err != nil {
				logger.Sugar().Errorf("download %s assert error %s", href, err)
				ec <- href
				return
			}
			f, err := os.Create(dir)
			if err != nil {
				logger.Sugar().Errorf("create file for  %s error: %v", href, err)
				ec <- href
				return
			}
			defer f.Close()
			_, err = io.Copy(f, res.Body) // 读取body的时间也算进httpclient的timeout里面了
			if err != nil {
				logger.Sugar().Errorf(" %s save2file  error: %v", href, err)
				ec <- href
				return
			}
			logger.Sugar().Infof("download resource  %s success", href)
			dc <- struct{}{}
		}(href, uri, jobCh, errCh, doneCh)
	})

	c.OnRequest(func(r *colly.Request) {
		logger.Sugar().Debugf("Visiting : %s", r.URL)
	})
	c.Visit("file://" + filepath) // http://go-colly.org/docs/introduction/start/

	var jc, ec, dc int32

	for {
		select {
		case <-jobCh:
			jc = jc + 1
		case e := <-errCh:
			ec = ec + 1
			failed = append(failed, e)
		case <-doneCh:
			dc = dc + 1
		default:
			if jc > 0 && jc == dc+ec {
				logger.Sugar().Infof("download %s assert resource failure: (%d), success (%d)", filepath, ec, dc)
				if len(failed) > 0 {
					return failed
				} else {
					return nil
				}
			}
		}
	}
}

// 制作一个时间戳
func makeTimeStamp(filepath string, failed []string) {
	f, err := os.Create(filepath)
	if err != nil {
		logger.Sugar().Errorf("create file error: %v", err)
		return
	}
	defer f.Close()
	io.Copy(f, bufio.NewReader(strings.NewReader(fmt.Sprintf("%s", failed))))
}

func html2md(filepath string) error {
	converter := md.NewConverter("", true, nil)

	f, _ := os.Open(filepath + ".html")
	markdown, err := converter.ConvertReader(f)
	if err != nil {
		logger.Sugar().Errorf("%s convert to markdown error: %v", filepath, err)
		return err
	}
	mdfile, err := os.Create(filepath + ".md")
	if err != nil {
		logger.Sugar().Errorf("create markdown file error: %v", err)
		return err
	}
	_, err = io.Copy(mdfile, &markdown)
	if err != nil {
		logger.Sugar().Errorf("write markdown file error: %v", err)
		return err
	}
	return nil
}
