package rpa

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"os"
	"sort"
	"time"
)

func OpenPage(browser *rod.Browser, url string, sleep int64, selector string, sign WaitSign) (page *rod.Page, err error) {
	page, err = browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return nil, err
	}

	err = page.WaitLoad()
	if err != nil {
		return nil, err
	}

	if selector != "" {
		if sign == WaitShow {
			err = WaitElementShow(page, selector, 20)
		} else if sign == WaitHide {
			err = WaitElementHide(page, selector, 20)
		}
	}

	if sleep > 0 {
		time.Sleep(time.Duration(sleep) * time.Second)
	}

	return
}

// WaitElementHide waiting for a certain element on the page to disappear
func WaitElementHide(page *rod.Page, selector string, timeoutSeconds int) (err error) {
	done := make(chan struct{}, 1)
	timeout := time.After(time.Second * time.Duration(timeoutSeconds))
	p := true
	v := true

	go func() {
		for {
			v = ElementVisible(page, selector)
			if !p && !v {
				break
			}
			if v != p {
				p = v
			}
			time.Sleep(time.Second)
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
		break
	case <-timeout:
		err = errors.New("wait element hide timed out")
		break
	}
	return
}

// WaitElementShow waiting for a certain element on the page to appear
func WaitElementShow(page *rod.Page, selector string, timeoutSeconds int) (err error) {
	done := make(chan struct{}, 1)
	timeout := time.After(time.Second * time.Duration(timeoutSeconds))
	p := false
	v := false
	go func() {
		for {
			v = ElementVisible(page, selector)
			if p && v {
				break
			}
			if v != p {
				p = v
			}
			time.Sleep(time.Second)
		}
		done <- struct{}{}
	}()

	select {
	case <-done:
		break
	case <-timeout:
		err = errors.New("wait element show timed out")
		break
	}

	return
}

// ElementVisible detects whether the selected element is existed and visible
func ElementVisible(page *rod.Page, selector string) bool {
	jsCode := fmt.Sprintf(`
		(selector) => {
            try {
                let elem = document.querySelector(selector);
                if(elem)
                    return elem.getBoundingClientRect().height > 0;
                else
                    return false;
            } catch(e){
                return false
            }
    	}`)

	result := page.MustEval(jsCode, selector)
	return result.Bool()
}

// WriteSortedJSONToFile writing indented and key sorted JSON to a file
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
	sortedJson, err := json.MarshalIndent(sortedData, "", "\t")
	if err != nil {
		return err
	}

	// write the json to a file
	return os.WriteFile(filename, sortedJson, 0644)
}

// import "github.com/AllenDang/w32"
//type Lan int32
//func (l Lan) Value() int32 {
//	return int32(l)
//}
//const (
//	EN Lan = 0x4090409
//	ZH     = 0x8040804
//)
//func ChangeLan(lang Lan) {
//	hwnd := w32.GetForegroundWindow()
//	// Language id for English
//	langId := lang.Value()
//	result := w32.SendMessage(hwnd, w32.WM_INPUTLANGCHANGEREQUEST, 0, langId)
//	if result == 0 {
//		// Success
//	} else {
//		// Failure
//	}
//}
