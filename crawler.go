package rpa

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-rod/rod/lib/input"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
)

type WaitSign string

const (
	WaitShow  WaitSign = "show"
	WaitHide  WaitSign = "hide"
	WaitDelay WaitSign = "wait"
)

type IPageLoad struct {
	Wait     WaitSign `json:"wait"`
	Selector string   `json:"selector,omitempty"`
	Sleep    int64    `json:"sleep,omitempty"`
}

type IConfigNode struct {
	Selector string `json:"selector"`
	Label    string `json:"label"`
	ID       string `json:"id"`
}

type DownloadType string

const (
	DownloadUrl     DownloadType = "url"
	DownloadElement DownloadType = "element"
)

type IDownloadConfig struct {
	IConfigNode
	SavePath   string       `json:"savePath,omitempty"`
	NameProper string       `json:"nameProper,omitempty"`
	Type       DownloadType `json:"type"`
}

type IConfig struct {
	PageLoad        IPageLoad         `json:"pageLoad,omitempty"`
	DataSection     []map[string]any  `json:"dataSection"`
	SwitchSection   map[string]any    `json:"switchSection,omitempty"`
	DownloadRoot    string            `json:"downloadRoot,omitempty"`
	DownloadSection []IDownloadConfig `json:"downloadSection,omitempty"`
}

// IDownloadResult is a part of result section
type IDownloadResult struct {
	Count     int      `json:"count"`
	Errors    []int    `json:"errors"`
	FileNames []string `json:"fileNames"`
	Links     []string `json:"links"`
}

type IExternalResult struct {
	Config  string `json:"config"`
	Connect string `json:"connect"`
	ID      string `json:"id"`
}

type IResult struct {
	Data            map[string]any             `json:"data"`
	DownloadRoot    string                     `json:"downloadRoot"`
	Downloads       map[string]IDownloadResult `json:"downloads"`
	ExternalSection map[string]IExternalResult `json:"externalSection"`
}

type Crawler struct {
	Browser    *rod.Browser
	CfgFetcher func(path string) (*IConfig, error)
}

func (c *Crawler) CrawlUrl(url string, cfgFilePath string, autoDownload bool, closeTab bool) (*IResult, *rod.Page, error) {
	_, cfg, err := c.fetchCfg(cfgFilePath)
	if err != nil {
		return nil, nil, err
	}

	wait := cfg.PageLoad.Wait
	selector := cfg.PageLoad.Selector
	delay := cfg.PageLoad.Sleep

	page, err := c.OpenPage(url, delay, selector, wait)
	if err != nil {
		return nil, nil, err
	}
	res, err := c.CrawlPage(page, cfgFilePath, autoDownload, closeTab)
	return res, page, err
}

func (c *Crawler) CrawlPage(page *rod.Page, cfgFilePath string, autoDownload bool, closeTab bool) (*IResult, error) {
	cfgBytes, cfg, err := c.fetchCfg(cfgFilePath)
	if err != nil {
		return nil, err
	}
	jsCode := fmt.Sprintf(`
	()=>{
		%s;
		return run(%s);
	}`, crawlerJs, cfgBytes)

	resultJson, err := page.Eval(jsCode)
	if err != nil {
		return nil, err
	}

	var result IResult
	err = resultJson.Value.Unmarshal(&result)
	if err != nil {
		return nil, err
	}

	if autoDownload && cfg.DownloadSection != nil && result.Downloads != nil {
		dlsMap := result.Downloads
		downloadRoot := result.DownloadRoot
		for _, dlCfgItem := range cfg.DownloadSection {
			key := dlCfgItem.ID
			if dlDataItem, ok := dlsMap[key]; ok {
				_ = c.download(page, dlCfgItem, &dlDataItem, downloadRoot)
			}
		}
	}

	if result.ExternalSection != nil {
		for _, extItem := range result.ExternalSection {
			if extItem.Config != "" {
				extCfg, _ := joinPath(cfgFilePath, extItem.Config)
				cc := extItem.Connect
				parts := strings.Split(cc, "/")
				secName := parts[0]
				itemName := parts[1]
				var resNode any
				resNode = result.Data
				if secName != "" {
					if m, ok := resNode.(map[string]any); ok {
						resNode = m[secName]
					} else {
						return nil, err
					}
				}

				if resNode != nil {
					if reflect.ValueOf(resNode).Kind() == reflect.Slice {
						if arr, ok := resNode.([]any); ok {
							for _, resExtNode := range arr {
								if extNode, oke := resExtNode.(map[string]any); oke {
									extUrl := extNode[itemName].(string)
									extData, _, err2 := c.CrawlUrl(extUrl, extCfg, autoDownload, closeTab)
									if err2 != nil {
										extNode[itemName] = fmt.Sprintf("an error occurred when crawling the external url: %s", extUrl)
									} else {
										extNode[itemName] = extData.Data
									}
								}
							}
						}
					} else {
						var extUrl string
						if m, ok := resNode.(map[string]interface{}); ok {
							extUrl, _ = m[itemName].(string)
							if extUrl != "" {
								extData, _, err2 := c.CrawlUrl(extUrl, extCfg, autoDownload, closeTab)
								if err2 != nil {
									return nil, err2
								}
								if s, oks := resNode.(map[string]interface{}); oks {
									s[itemName] = extData.Data
								} else {
									return nil, err
								}
							}
						} else {
							return nil, errors.New(fmt.Sprintf("parse %s node error", itemName))
						}
					}
				}
			}
		}
	}

	if closeTab {
		_ = page.Close()
	}

	return &result, nil
}

func (c *Crawler) AttachDefaultBrowser() *rod.Browser {
	wsURL := launcher.NewUserMode().MustLaunch()
	c.Browser = rod.New().ControlURL(wsURL).MustConnect().NoDefaultDevice()
	return c.Browser
}

func (c *Crawler) OpenPage(url string, sleep int64, selector string, sign WaitSign) (page *rod.Page, err error) {
	page, err = c.Browser.Page(proto.TargetCreateTarget{URL: url})
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

func (c *Crawler) download(page *rod.Page, dlCfg IDownloadConfig, dlData *IDownloadResult, downloadRoot string) error {
	selector := dlCfg.Selector
	downType := dlCfg.Type

	var subDir string
	if len(dlCfg.SavePath) > 0 {
		subDir = dlCfg.SavePath
	} else {
		subDir = dlCfg.ID
	}
	saveDir := path.Join(downloadRoot, subDir)
	//err := os.MkdirAll(saveDir, os.ModePerm)
	//if err != nil {
	//	return err
	//}

	browser := page.Browser()
	elems, err := page.Elements(selector)
	if err != nil {
		return err
	}

	for i, elem := range elems {
		fileFullPathName := path.Join(saveDir, dlData.FileNames[i])
		waitDownload := browser.MustWaitDownload()
		if downType == DownloadUrl {
			_ = page.Keyboard.Press(input.AltLeft)
		}
		elem.MustClick()
		err = utils.OutputFile(fileFullPathName, waitDownload())
		if err != nil {
			dlData.Errors = append(dlData.Errors, i)
		}
		if downType == DownloadUrl {
			_ = page.Keyboard.Release(input.AltLeft)
		}
	}

	return nil
}

func (c *Crawler) fetchCfg(cfgPath string) ([]byte, *IConfig, error) {
	if c.CfgFetcher != nil {
		cfg, err := c.CfgFetcher(cfgPath)
		if err != nil {
			return nil, nil, err
		} else {
			cfgBytes, err2 := json.Marshal(cfg)
			return cfgBytes, cfg, err2
		}
	} else {
		return innerFetcher(cfgPath)
	}
}

func innerFetcher(cfgFilePath string) ([]byte, *IConfig, error) {
	cfgJsonStr, err := os.ReadFile(cfgFilePath)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		return nil, nil, err
	}
	var cfg IConfig
	err = json.Unmarshal(cfgJsonStr, &cfg)
	if err != nil {
		return nil, nil, err
	}
	return cfgJsonStr, &cfg, nil
}

func joinPath(p1, p2 string) (string, error) {
	match, _ := regexp.MatchString("^https?://", p1)
	if match {
		base, err := url.Parse(p1)
		if err != nil {
			return "", err
		}
		ref, err := url.Parse(p2)
		if err != nil {
			return "", err
		}
		return base.ResolveReference(ref).String(), nil
	} else {
		if filepath.IsAbs(p2) {
			return p2, nil
		}
		p1 = filepath.Dir(p1)
		p := filepath.Join(p1, p2)
		return filepath.Abs(p)
	}

}

//go:embed "resource/crawler.js"
var crawlerJs string
