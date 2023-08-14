package rpa

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

// ConnectDefaultBrowser returns the system's default browser
func ConnectDefaultBrowser(leakless, headless bool) (br *rod.Browser, err error) {
	wsURL, err := launcher.NewUserMode().Leakless(leakless).Headless(headless).Launch()
	if err != nil {
		return
	}
	br = rod.New().ControlURL(wsURL).NoDefaultDevice()
	err = br.Connect()
	if err != nil {
		return
	}
	return br, err
}

// ConnectChromiumBrowser returns the rod's embed browser
func ConnectChromiumBrowser(leakless, headless bool) (br *rod.Browser, err error) {
	l := launcher.New().Leakless(leakless).Headless(headless)
	wsURL, err := l.Launch()
	if err != nil {
		return
	}

	br = rod.New().ControlURL(wsURL)
	err = br.Connect()
	if err != nil {
		return
	}
	return br, err
}

// ConnectChromeBrowser returns the Chrome browser if installed
func ConnectChromeBrowser(leakless, headless bool) (br *rod.Browser, err error) {
	chrome, found := launcher.LookPath()
	if !found {
		err = errors.New("chrome path not found")
		return
	}

	l := launcher.New().Bin(chrome).Leakless(leakless).Headless(headless).
		Set("disable-default-apps").
		Set("no-first-run").
		Set("no-default-browser-check")

	wsURL, err := l.Launch()
	if err != nil {
		return
	}

	br = rod.New().ControlURL(wsURL)
	err = br.Connect()
	if err != nil {
		return
	}
	return br, err
}

// ConnectEdgeBrowser returns the Edge browser if installed
func ConnectEdgeBrowser(leakless, headless bool, ieMode bool) (br *rod.Browser, err error) {
	p := "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe"
	_, err = os.Stat(p)
	if os.IsNotExist(err) {
		err = errors.New("edge path not found")
		return
	}

	l := launcher.New().Bin(p).Leakless(leakless).Headless(headless).
		Set("disable-default-apps").
		Set("no-first-run").
		Set("no-default-browser-check")

	if ieMode {
		l.Set("--ie-mode-force").
			Set("--internet-explorer-integration", "iemode").
			Set("--no-service-autorun").
			Set("--disable-sync").
			Set("--disable-features", "msImplicitSignin")
		//.Delete("--remote-debugging-port")
	}

	wsURL, err := l.Launch()
	if err != nil {
		return
	}

	br = rod.New().ControlURL(wsURL)
	err = br.Connect()
	if err != nil {
		return
	}

	return br, err
}

func OpenPage(browser *rod.Browser, url string, sleep int64, selector string, sign WaitSign) (page *rod.Page, err error) {
	page, err = browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return nil, err
	}
	err = WaitPage(page, sleep, selector, sign)
	return page, err
}

func WaitPage(page *rod.Page, sleep int64, selector string, sign WaitSign) (err error) {
	err = page.WaitStable(time.Second * 30)
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

func RaceShow(page *rod.Page, selectors []string, timeoutSeconds int) (index int, elem *rod.Element, err error) {
	done := make(chan *rod.Element, 1)
	timeout := time.After(time.Second * time.Duration(timeoutSeconds))
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
	OutLoop:
		for {
			for i, selector := range selectors {
				if ElementVisible(page, selector) {
					index = i
					done <- page.MustElement(selector)
					break OutLoop
				}
			}
			time.Sleep(time.Millisecond * 100)
		}
	}()

	select {
	case elem = <-done:
		break
	case <-timeout:
		err = errors.New("wait elements timed out")
		break
	}
	return
}

// WaitElementHide waiting for a certain element on the page to disappear
func WaitElementHide(page *rod.Page, selector string, timeoutSeconds int) (err error) {
	v := ElementVisible(page, selector)
	if !v {
		return
	}
	p := true
	done := make(chan struct{}, 1)
	timeout := time.After(time.Second * time.Duration(timeoutSeconds))

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
			v = ElementVisible(page, selector)
			if !p && !v {
				break
			}
			if v != p {
				p = false
			}
			time.Sleep(time.Millisecond * 100)
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
	v := ElementVisible(page, selector)
	if v {
		return
	}
	p := false
	done := make(chan struct{}, 1)
	timeout := time.After(time.Second * time.Duration(timeoutSeconds))
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
			v = ElementVisible(page, selector)
			if p && v {
				break
			}
			if v != p {
				p = true
			}
			time.Sleep(time.Millisecond * 100)
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
					doc = pseudoType === 'frame' ? pseudoElem.contentWindow.document : pseudoElem.shadowRoot;
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
		} catch (e) {
			return false;
		}
	}`)

	result := page.MustEval(jsCode, selector)
	return result.Bool()
}

func QueryElem(page *rod.Page, selector string) (*rod.Element, error) {
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
					doc = pseudoType === 'frame' ? pseudoElem.contentWindow.document : pseudoElem.shadowRoot;
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
				return elem;
			} else {
				return null;
			}
		} catch (e) {
			return null;
		}
	}`)

	opts := &rod.EvalOptions{
		JS: jsCode,
		JSArgs: []interface{}{
			selector,
		},
	}
	return page.ElementByJS(opts)
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

// EmptyDirectory will delete all the contents of a directory
func EmptyDirectory(dir string) error {
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
