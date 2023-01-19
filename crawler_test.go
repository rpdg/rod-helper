package rpa

import (
	"fmt"
	"os"
	"testing"
)

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

func TestCrawler_CrawlUrl(t *testing.T) {
	t.Run("crawl by url", func(t *testing.T) {

		testUrl := "https://cn.bing.com/search?q=sample+simple+pdf"
		testFile := "./bing.json"
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
				if list, ok2 := data.([]any); ok2 {
					expected := 10
					l := len(list)
					if l != expected {
						t.Errorf("expected %d but got %d", expected, l)
					}
					fmt.Println()
				}
			}
		}

	})
}
