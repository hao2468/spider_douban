package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

// var url string = "https://movie.douban.com/tv/#!type=tv&tag=%E6%97%A5%E6%9C%AC%E5%8A%A8%E7%94%BB&sort=recommend&page_limit=20&page_start=0"
// var dir string = "anime_douban.txt"
// var target_grade int = 9

var old string
var new string

var newdoc *goquery.Document
var endflag bool = true

type Cfg struct {
	Filepath string `json:"filepath"`
	Url      string `json:"url"`
	Grade    int    `json:"grade"`
}

func main() {
	cfgdata, err := ioutil.ReadFile("cfg.json")
	if err != nil {
		fmt.Println("打开配置文件失败")
		return
	}
	cfg := Cfg{}
	err = json.Unmarshal(cfgdata, &cfg)
	if err != nil {
		fmt.Println("解析配置文件失败")
		return
	}
	fmt.Println("cfg:", cfg)
	endch := make(chan int, 1)
	outputch := make(chan string, 10)
	go ender(endch)
	parentctx := context.Background()
	ctx, cancel := chromedp.NewContext(parentctx, chromedp.WithLogf(log.Printf))
	defer cancel()
	go outputtofile(ctx, cfg.Filepath, outputch)
	chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(cfg.Url),
	})
	var doc *goquery.Document
	for endflag {
		fmt.Println("加载新的动态页面...")
		for i := 0; i < 10; i++ {
			var str string
			ticker := time.NewTicker(5 * time.Second)
			runch := make(chan int, 1)
			go chromedp.Run(ctx, clickmore(&str, runch))
			select {
			case <-runch:
			case <-ticker.C:
				endch <- 0
				goto run
			}
			newdoc, err := goquery.NewDocumentFromReader(strings.NewReader(str))
			if err != nil {
				fmt.Println("创建完整doc错误:", err)
				return
			}
			doc = newdoc
		}
	run:
		new, err := doc.Find(".list-wp").Find(".list").Html()
		if err != nil {
			fmt.Println("转换html错误:", err)
			return
		}

		var clearstr string
		if old == "" {
			clearstr = new
		} else {
			clearstr = strings.Replace(new, old, "", -1)
		}
		newdoc, err := goquery.NewDocumentFromReader(strings.NewReader(clearstr))
		if err != nil {
			fmt.Println("创建去重doc错误:", err)
			return
		}
		fmt.Println("正在解析并写入...")
		parseWeb(newdoc, outputch, cfg.Grade)
		old = new
	}
	ticker := time.NewTicker(5 * time.Second)
	<-ticker.C
	endch <- 1
	fmt.Println("退出")
}

func clickmore(res *string, runch chan int) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Sleep(500 * time.Millisecond),
		chromedp.Click(".more", chromedp.ByQuery),
		chromedp.OuterHTML(`body`, res, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			runch <- 0
			return nil
		}),
	}
}

func parseWeb(doc *goquery.Document, ch chan string, targetGrade int) {
	doc.Find("a[class=item]").Each(func(i int, s *goquery.Selection) {
		grade := s.Find("p").Find("strong").Remove().Text()
		if gradefloat, _ := strconv.ParseFloat(grade, 32); int(gradefloat) > targetGrade {
			name := strings.Replace(s.Find("p").Text(), "\n", "", -1)
			name = strings.Replace(name, " ", "", -1)
			str := fmt.Sprintln(name, "	评分:", grade)
			href, exist := s.Attr("href")
			if exist {
				str += fmt.Sprintln("url=", href)
			}
			ch <- str
		}
	})
}

func outputtofile(ctx context.Context, filepath string, ch chan string) {
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE, 0744)
	defer file.Close()
	if err != nil {
		panic("打开文件失败")
	}
	for {
		select {
		case str := <-ch:
			fmt.Fprintln(file, str)
		case <-ctx.Done():
			return
		}
	}
}

func ender(endch chan int) {
	for {
		select {
		case i := <-endch:
			if i == 1 {
				return
			}
			fmt.Println("爬取完成,等待退出")
			endflag = false
		}
	}
}
