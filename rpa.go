package rpa

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rpdg/rod-helper/resource"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
)

type WaitTarget string

const (
	WaitShow  WaitTarget = "show"
	WaitHide  WaitTarget = "hide"
	WaitDelay WaitTarget = "wait"
)

type IPageLoad struct {
	Wait     WaitTarget `json:"wait"`
	Selector string     `json:"selector,omitempty"`
	Sleep    int64      `json:"sleep,omitempty"`
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

type RPA struct {
	Browser    *rod.Browser
	CfgFetcher func(path string) (*IConfig, error)
}

func (r *RPA) CrawlUrl(url string, cfgFilePath string, autoDownload bool, closeTab bool) (*IResult, *rod.Page, error) {
	_, cfg, err := r.fetchCfg(cfgFilePath)
	if err != nil {
		return nil, nil, err
	}

	wait := cfg.PageLoad.Wait
	selector := cfg.PageLoad.Selector
	delay := cfg.PageLoad.Sleep

	page, err := r.OpenPage(url, delay, selector, wait)
	if err != nil {
		return nil, nil, err
	}
	res, err := r.CrawlPage(page, cfgFilePath, autoDownload, closeTab)
	return res, page, err
}

func (r *RPA) CrawlPage(page *rod.Page, cfgFilePath string, autoDownload bool, closeTab bool) (*IResult, error) {
	cfgBytes, cfg, err := r.fetchCfg(cfgFilePath)
	if err != nil {
		return nil, err
	}
	jsCode := fmt.Sprintf(`
	()=>{
		%s;
		return run(%s);
	}`, string(resource.CrawlerJs), cfgBytes)

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
				_ = r.download(page, dlCfgItem, &dlDataItem, downloadRoot)
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
					if m, ok := resNode.(map[string]interface{}); ok {
						resNode = m[secName]
					} else {
						return nil, err
					}
				}

				var extUrl string
				if m, ok := resNode.(map[string]interface{}); ok {
					extUrl, _ = m[itemName].(string)
				} else {
					return nil, err
				}

				if extUrl != "" {
					extData, _, err2 := r.CrawlUrl(extUrl, extCfg, autoDownload, closeTab)
					if err2 != nil {
						return nil, err2
					}

					if m, ok := resNode.(map[string]interface{}); ok {
						m[itemName] = extData.Data
					} else {
						return nil, err
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

func (r *RPA) AttachDefaultBrowser() *rod.Browser {
	wsURL := launcher.NewUserMode().MustLaunch()
	r.Browser = rod.New().ControlURL(wsURL).MustConnect().NoDefaultDevice()
	return r.Browser
}

func (r *RPA) OpenPage(url string, sleep int64, selector string, sign WaitTarget) (page *rod.Page, err error) {
	page = r.Browser.MustPage(url)
	page.MustWaitLoad()

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

func (r *RPA) download(page *rod.Page, dlCfg IDownloadConfig, dlData *IDownloadResult, downloadRoot string) error {
	selector := dlCfg.Selector
	downType := dlCfg.Type

	var subDir string
	if len(dlCfg.SavePath) > 0 {
		subDir = dlCfg.SavePath
	} else {
		subDir = dlCfg.ID
	}
	saveDir := path.Join(downloadRoot, subDir)
	err := os.MkdirAll(saveDir, os.ModePerm)
	if err != nil {
		return err
	}

	for i, fileName := range dlData.FileNames {
		fileFullPathName := path.Join(saveDir, fileName)
		if downType == DownloadElement {
			browser := page.Browser()
			elems, err := page.Elements(selector)
			if err != nil {
				return err
			}
			waitDownload := browser.MustWaitDownload()
			elems[i].MustClick()
			err = utils.OutputFile(fileFullPathName, waitDownload())
			if err != nil {
				dlData.Errors = append(dlData.Errors, i)
			}
		} else if downType == DownloadUrl {
			fileBytes, err := page.GetResource(dlData.Links[i])
			if err != nil {
				dlData.Errors = append(dlData.Errors, i)
			} else {
				err := utils.OutputFile(fileFullPathName, fileBytes)
				if err != nil {
					dlData.Errors = append(dlData.Errors, i)
				}
			}
		}
	}

	return nil
}

func (r *RPA) fetchCfg(cfgPath string) ([]byte, *IConfig, error) {
	if r.CfgFetcher != nil {
		cfg, err := r.CfgFetcher(cfgPath)
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
	if filepath.IsAbs(p2) {
		return p2, nil
	}
	p1 = filepath.Dir(p1)
	p := filepath.Join(p1, p2)
	return filepath.Abs(p)
}
