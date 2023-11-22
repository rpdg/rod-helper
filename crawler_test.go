package rpa

import (
	"encoding/json"
	"os"
	"testing"
)

func Test_CrawlUrl(t *testing.T) {
	t.Run("crawl bing.com and download PDF files", func(t *testing.T) {
		testUrl := "https://www.learningcontainer.com/sample-zip-files/"
		r := Crawler{}
		err := r.AttachEmbedBrowser()
		if err != nil {
			t.Errorf("connect browser failed: %v", err)
			return
		}
		val, _, err := r.CrawlUrl(testUrl, "./sample/sample_zip.json", true, true)
		if err != nil {
			t.Errorf("crawler failed: %v", err)
			return
		}
		if _, ok := val.Data["linkList"]; ok {
			b, _ := json.MarshalIndent(val, "", "\t")
			err2 := os.WriteFile("./zip_result.json", b, 0644)
			if err2 != nil {
				t.Errorf("%v", err2)
			}
		}

	})
}
