package rpa

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"
)

func OpenPage(browser *rod.Browser, url string, sleep int64, selector string, sign WaitSign) (page *rod.Page, err error) {
	page, err = browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return nil, err
	}
	err = WaitPage(page, sleep, selector, sign)
	return page, err
}

func WaitPage(page *rod.Page, sleep int64, selector string, sign WaitSign) (err error) {
	err = page.WaitLoad()
	if err != nil {
		return err
	}

	err = page.WaitIdle(time.Second * 30)
	if err != nil {
		return err
	}

	if selector != "" {
		switch sign {
		case WaitShow:
			err = WaitElementShow(page, selector, 20)
		case WaitHide:
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
	var mu sync.Mutex
	done := make(chan struct{}, 1)
	timeout := time.After(time.Second * time.Duration(timeoutSeconds))
	p := true
	v := true

	go func() {
		defer func() {
			if r := recover(); r != nil {
				switch x := r.(type) {
				case string:
					err = errors.New(x)
				case error:
					err = x
				default:
					err = errors.New("unknown panic")
				}
			}
		}()
		for {
			mu.Lock()
			v = ElementVisible(page, selector)
			if !p && !v {
				mu.Unlock()
				break
			}
			if v != p {
				p = v
			}
			mu.Unlock()
			time.Sleep(time.Millisecond * 200)
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
	var mu sync.Mutex
	done := make(chan struct{}, 1)
	timeout := time.After(time.Second * time.Duration(timeoutSeconds))
	p := false
	v := false
	go func() {
		defer func() {
			if r := recover(); r != nil {
				switch x := r.(type) {
				case string:
					err = errors.New(x)
				case error:
					err = x
				default:
					err = errors.New("unknown panic")
				}
			}
		}()
		for {
			mu.Lock()
			v = ElementVisible(page, selector)
			if p && v {
				mu.Unlock()
				break
			}
			if v != p {
				p = v
			}
			mu.Unlock()
			time.Sleep(time.Millisecond * 200)
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
			function replacePseudo(selector, parentElement = document) {
				let doc = parentElement;
				let ctxChanged = false;
				let pseudoMatch = selector.match(/^:(frame|shadow)\((.+?)\)/);
				if (pseudoMatch) {
					let pseudoType = pseudoMatch[1];
					let pseudoSelector = pseudoMatch[2];
					let pseudoElem = parentElement.querySelector(pseudoSelector);
					if (pseudoElem) {
						doc =
							pseudoType === 'frame'
								? pseudoElem.contentWindow.document
								: pseudoElem.shadowRoot;
						selector = selector.slice(pseudoMatch[0].length).trim();
						ctxChanged = true;
					}
				}
				if (/^:(frame|shadow)\(/.test(selector)) {
					return replacePseudo(selector, doc);
				}
				return { doc, selector, ctxChanged };
			}
			function queryElem(selectorString, parentElement = document) {
				let secNode = null;
				let { doc, selector } = replacePseudo(selectorString, parentElement);
				secNode = doc.querySelector(selector);
				return secNode;
			}
            try {
                let elem = queryElem(selector);
                if (elem) {
                    let rect = elem.getBoundingClientRect();
                    return rect.height > 0 && rect.width > 0;
                } else {
                    return false;
				}
            } catch(e) {
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

// RenameFileUnique rename file name if there are duplicate files
func RenameFileUnique(dir, fileName, ext string, try int) string {
	var rawFileName string
	if try == 0 {
		rawFileName = path.Join(dir, fmt.Sprintf("%s%s", fileName, ext))
	} else {
		rawFileName = path.Join(dir, fmt.Sprintf("%s_%d%s", fileName, try, ext))
	}

	if exists, _ := FileExists(rawFileName); exists {
		return RenameFileUnique(dir, fileName, ext, try+1)
	}

	return rawFileName
}

// FileExists to check if a file exists
func FileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// RemoveContents will delete all the contents of a directory
func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

var exp = regexp.MustCompile(`([<>:"/\\\|?*]+)`)

// NormalizeFilename will replace <>:"/\|?* in string
func NormalizeFilename(name string) string {
	outName := exp.ReplaceAllString(name, "_")
	//println(name, outName)
	return outName
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
