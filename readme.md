## spider用于爬取豆瓣热门电视剧/电影页面指定评分的信息，记录信息包括影片名/剧名，评分以及其豆瓣介绍页面url

### 快速使用

在程序exe同目录下创建cfg.json，填入需要爬取的url，爬取信息存储位置，指定爬取的评分

#### cfg样例

```json
{

  "filepath":"anime_douban.txt",

  "url":"https://movie.douban.com/tv/#!type=tv&tag=%E6%97%A5%E6%9C%AC%E5%8A%A8%E7%94%BB&sort=recommend&page_limit=20&page_start=0",

  "grade":9

}
```

本项目依赖于`github.com/chromedp/chromedp`，该模块用于模拟谷歌浏览器浏览动作，程序运行环境需包含谷歌浏览器或安装headless-shell