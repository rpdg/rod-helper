package rpa

import (
	"encoding/json"
	"os"
	"testing"
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

func Test_K2(t *testing.T) {
	t.Run("crawl k2 list page", func(t *testing.T) {
		url2 := "https://flowcenter.opg.cn/Portal/ProcessCenter/MyFlowList"
		c := Crawler{}
		c.AttachDefaultBrowser()
		val, _, err := c.CrawlUrl(url2, "./sample/k2_list.json", true, true)
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
