package rpa

import (
	"context"
	"errors"
	"fmt"
	"github.com/axgle/mahonia"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ConnectDefaultBrowser returns the system's default browser
func ConnectDefaultBrowser(leakless, headless bool) (br *rod.Browser, err error) {
	wsURL, err := launcher.NewUserMode().Leakless(leakless).Headless(headless).
		Set("disable-default-apps").
		Set("no-first-run").
		Set("no-default-browser-check").Launch()
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
	l := launcher.New().Leakless(leakless).Headless(headless).
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
	err = page.WaitStable(time.Second)
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

// GetDictAndLastSegmentByPath returns the data extracted from the path and the last segment of the path.
func GetDictAndLastSegmentByPath(data map[string]interface{}, path string) (interface{}, string) {
	keys := strings.Split(strings.Trim(path, "/"), "/")
	lastSegment := keys[len(keys)-1]

	var dictData interface{} = data
	for _, key := range keys[:len(keys)-1] {
		if dict, ok := dictData.(map[string]interface{}); ok {
			if val, exists := dict[key]; exists {
				dictData = val
			} else {
				dictData = nil
				break
			}
		} else {
			dictData = nil
			break
		}
	}

	return dictData, lastSegment
}

type ExecuteResult struct {
	output string
	err    error
}

// ExecShell 执行shell命令，可设置执行超时时间
func ExecShell(ctx context.Context, command string) (string, error) {
	cmd := exec.Command("cmd", "/C", command)
	// 隐藏cmd窗口
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	var resultChan chan ExecuteResult = make(chan ExecuteResult)
	go func() {
		output, err := cmd.CombinedOutput()
		resultChan <- ExecuteResult{string(output), err}
	}()
	select {
	case <-ctx.Done():
		if cmd.Process.Pid > 0 {
			exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
			cmd.Process.Kill()
		}
		return "", errors.New("timeout killed")
	case result := <-resultChan:
		return GBK2UTF8(result.output), result.err
	}
}

// GBK2UTF8 GBK编码转换为UTF8
func GBK2UTF8(s string) string {
	dec := mahonia.NewDecoder("gbk")
	return dec.ConvertString(s)
}

func ExtractUrlParam(urlString, paramName string) (string, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return "", err
	}
	params, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return "", err
	}
	values, ok := params[paramName]
	if !ok || len(values) == 0 {
		return "", fmt.Errorf("parameter not found")
	}
	return values[0], nil
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
