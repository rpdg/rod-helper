package rpa

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
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

func Test_K2(t *testing.T) {
	url1 := "https://flowcenter.opg.cn/Procmanage/flowmanage?ProcInstID=61336"
	c := Crawler{}
	c.AttachDefaultBrowser()
	val, _, err := c.CrawlUrl(url1, "./sample/k2_d1.json", false, true)
	if err != nil {
		fmt.Println("err: ", err)
	} else {
		if val == nil {
			t.Errorf("nil result")
		} else {
			//s, _ := json.MarshalIndent(val, "", "\t")
			//fmt.Println(string(s))
			WriteSortedJSONToFile(val, "./res.json")
		}
	}
}

func WriteSortedJSONToFile(data interface{}, filename string) error {
	// marshal the struct to json
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	var jsonData map[string]interface{}
	err = json.Unmarshal(b, &jsonData)
	if err != nil {
		return err
	}

	// sort the keys
	var keys []string
	for key := range jsonData {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// rebuild the json object in sorted order
	sortedData := make(map[string]interface{})
	for _, key := range keys {
		sortedData[key] = jsonData[key]
	}
	// marshal the sorted json object
	sortedJson, err := json.Marshal(sortedData)
	if err != nil {
		return err
	}

	// write the json to a file
	return os.WriteFile(filename, sortedJson, 0644)
}
