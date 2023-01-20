package rpa

import (
	"os"
	"testing"
)

func TestCrawler_CrawlUrl(t *testing.T) {
	t.Run("crawl bing.com and download PDF files", func(t *testing.T) {
		const sampleConfigJson = `
		  {
		  "dataSection": [
			{
			  "label": "Bing Search Results",
			  "id": "linkList",
			  "selector": "#b_results > .b_algo > .b_title > h2",
			  "sectionType": "list",
			  "items": [
				{
				  "label": "page link",
				  "id": "url",
				  "selector": "a",
				  "itemType": "text",
				  "valueProper": "href"
				},
				{
				  "label": "page title",
				  "id": "title",
				  "selector": "a",
				  "itemType": "text"
				}
			  ]
			}
		  ],
		  "downloadRoot": "c:\\attachment2",
		  "downloadSection": [
			{
			  "selector": "#b_results > .b_algo > .b_title > .sb_doct_txt.b_float + h2 > a",
			  "label": "PDF File",
			  "id": "files",
			  "nameProper": "href",
			  "nameRender": "let parts = name.split('/');return parts[parts.length - 1];",
			  "type": "url"
			}
		  ]
		}`
		testUrl := "https://cn.bing.com/search?q=sample+simple+pdf"
		testFile := "./bing.config.json"
		os.WriteFile(testFile, []byte(sampleConfigJson), 0644)
		defer func() {
			os.Remove(testFile)
		}()
		r := Crawler{}
		r.AttachDefaultBrowser()
		val, _, err := r.CrawlUrl(testUrl, testFile, true, true)
		if err != nil {
			t.Errorf("crawler failed: %v", err)
		} else {
			if data, ok := val.Data["linkList"]; ok {
				if list, ok2 := data.([]interface{}); ok2 {
					expected := 10
					l := len(list)
					if l != expected {
						t.Errorf("expected %d but got %d", expected, l)
					}
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
		val, _, err := c.CrawlUrl(url1, "./sample/k2_d1.json", false, true)
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
