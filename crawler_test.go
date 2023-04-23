package rpa

import (
	"encoding/json"
	"github.com/go-rod/rod"
	"os"
	"testing"
	"time"
)

func Test_CrawlUrl(t *testing.T) {
	t.Run("crawl bing.com and download PDF files", func(t *testing.T) {
		testUrl := "https://cn.bing.com/search?q=sample+simple+pdf"
		r := Crawler{}
		r.AttachDefaultBrowser()
		val, _, err := r.CrawlUrl(testUrl, "./sample/bing.json", false, true)
		if err != nil {
			t.Errorf("crawler failed: %v", err)
		} else {
			if _, ok := val.Data["linkList"]; ok {
				b, _ := json.MarshalIndent(val, "", "\t")
				err2 := os.WriteFile("./bing_result.json", b, 0644)
				if err2 != nil {
					t.Errorf("%v", err2)
				}
			}
		}

	})
}

func openNccTab2(page *rod.Page, t1, t2 string) (newPage *rod.Page) {

	page.MustElement(".nc-workbench-allAppsBtn").MustClick()
	time.Sleep(time.Millisecond * 200)

	items := page.MustElements(".sider .result-group-list .list-item-content")
	for _, item := range items {
		txt := item.MustText()
		if txt == t1 {
			item.MustClick()
			time.Sleep(time.Millisecond * 200)
			links := page.MustElements(".content .content-item .item-app")
			for _, link := range links {
				linkText := link.MustText()
				if linkText == t2 {
					wait := page.MustWaitOpen()
					link.MustClick()
					newPage = wait()
					newPage.MustWaitLoad()
					iframe := newPage.MustElement("iframe").MustFrame()
					iframe.MustWaitRequestIdle()
					return
				}
			}
			break
		}
	}

	return nil
}

func Test_NCC(t *testing.T) {
	t.Run("ncc iframe", func(t *testing.T) {
		c := Crawler{}
		c.AttachDefaultBrowser()
		page := c.Browser.MustPage("http://10.33.33.66:8090/nccloud/resources/workbench/public/common/main/index.html#/")
		defer page.MustClose()
		p2 := openNccTab2(page, "组织管理", "集团")
		val, err := c.CrawlPage(p2, "./sample/ncc_data.json", false, true)
		if err != nil {
			t.Errorf("%v", err)
		} else {
			if val == nil {
				t.Errorf("nil result")
			} else {
				err2 := WriteSortedJSONToFile(val, "./res_ncc.json")
				if err2 != nil {
					t.Errorf("%v", err2)
				}
			}
		}
	})
}
func Test_K2(t *testing.T) {
	t.Run("crawl k2 list page", func(t *testing.T) {
		url2 := "https://flowcenter.opg.cn/Portal/ProcessCenter/MyFlowList"
		c := Crawler{}
		c.AttachDefaultBrowser()
		val, _, err := c.CrawlUrl(url2, "./sample/k2_list.json", false, true)
		if err != nil {
			t.Errorf("%v", err)
		} else {
			if val == nil {
				t.Errorf("nil result")
			} else {
				err := WriteSortedJSONToFile(val, "./res_list.json")
				if err != nil {
					t.Errorf("%v", err)
				}
			}
		}
	})
	t.Run("crawl k2 main page", func(t *testing.T) {
		url1 := "https://flowcenter.opg.cn/Procmanage/flowmanage?ProcInstID=61336"
		c := Crawler{}
		c.AttachDefaultBrowser()
		val, _, err := c.CrawlUrl(url1, "./sample/k2_d1.json", true, true)
		if err != nil {
			t.Errorf("%v", err)
		} else {
			if val == nil {
				t.Errorf("nil result")
			} else {
				err := WriteSortedJSONToFile(val, "./res.json")
				if err != nil {
					t.Errorf("%v", err)
				}
			}
		}
	})
}
