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

func TestAsyncFunction(t *testing.T) {
	r := Crawler{}
	r.AttachDefaultBrowser()
	page := r.Browser.MustPage("https://www.baidu.com")

	t.Log("sl")

	resultJson, err := page.Eval(`
		async function main(){
            function sleep(t , v) {
                return new Promise(function(resolve) {
                    setTimeout(resolve.bind(null, v), t);
                });
            }
            await sleep(3e3);
            return 333;
        }
	`)
	if err != nil {
		t.Errorf(err.Error())
	} else {
		t.Log(resultJson)
	}

}
