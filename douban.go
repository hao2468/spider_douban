package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

var old string
var new string

var newdoc *goquery.Document
var endflag bool = true

type Cfg struct {
	Filepath string  `json:"filepath"`
	Url      string  `json:"url"`
	Grade    float32 `json:"grade"`
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
	fmt.Println("开始爬取...")
	endch := make(chan int, 1)
	outputch := make(chan string, 10)
	go ender(endch)
	parentctx := context.Background()
	sysType := runtime.GOOS
	var ctx context.Context
	if sysType == "linux" {
		opts := []chromedp.ExecAllocatorOption{
			chromedp.Flag("headless", false),
			chromedp.UserAgent(`Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36`),
		}
		opts = append(chromedp.DefaultExecAllocatorOptions[:], opts...)
		parentctx, _ = chromedp.NewExecAllocator(parentctx, opts...)
	}
	ctx, cancel := chromedp.NewContext(parentctx, chromedp.WithLogf(log.Printf))
	defer cancel()
	go outputtofile(ctx, cfg.Filepath, outputch, endch)
	err = chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Navigate(cfg.Url),
	})
	if err != nil {
		fmt.Println("创建chrome实例失败，error:", err)
		endch <- -1
		return
	}
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
	fmt.Println("爬取已完成，退出")
}

func clickmore(res *string, runch chan int) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Sleep(200 * time.Millisecond),
		chromedp.Click(".more", chromedp.ByQuery),
		chromedp.OuterHTML(`body`, res, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			runch <- 0
			return nil
		}),
	}
}

func parseWeb(doc *goquery.Document, ch chan string, targetGrade float32) {
	doc.Find("a[class=item]").Each(func(i int, s *goquery.Selection) {
		grade := s.Find("p").Find("strong").Remove().Text()
		if gradefloat, _ := strconv.ParseFloat(grade, 32); float32(gradefloat) > targetGrade {
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

func outputtofile(ctx context.Context, filepath string, ch chan string, enderch chan int) {
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0744)
	defer file.Close()
	if err != nil {
		panic("打开文件失败")
	}
	for {
		select {
		case str := <-ch:
			_, err := fmt.Fprintln(file, str)
			if err != nil {
				fmt.Println("文件写入失败,error:", err)
				enderch <- -1
			}
		case <-ctx.Done():
			return
		}
	}
}

func ender(endch chan int) {
	switch <-endch {
	case 1:
		return
	case 0:
		fmt.Println("爬取完成,等待文件写入完成...")
		endflag = false
		return
	case -1:
		fmt.Println("爬取失败,等待程序退出...")
		endflag = false
		return
	}
}
