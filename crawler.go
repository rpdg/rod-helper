package rpa

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/utils"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	PageLoad        IPageLoad                `json:"pageLoad,omitempty"`
	DataSection     []map[string]interface{} `json:"dataSection"`
	SwitchSection   map[string]interface{}   `json:"switchSection,omitempty"`
	DownloadRoot    string                   `json:"downloadRoot,omitempty"`
	DownloadSection []IDownloadConfig        `json:"downloadSection,omitempty"`
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
	Data            map[string]interface{}     `json:"data"`
	DownloadRoot    string                     `json:"downloadRoot"`
	Downloads       map[string]IDownloadResult `json:"downloads"`
	ExternalSection map[string]IExternalResult `json:"externalSection"`
}

type Crawler struct {
	Browser    *rod.Browser
	CfgFetcher func(path string) (*IConfig, error)
}

func (c *Crawler) CrawlUrl(url string, cfgOrFile interface{}, autoDownload bool, closeTab bool) (*IResult, *rod.Page, error) {
	var cfg *IConfig
	var err error
	cfgFilePath := ""
	switch cfgOrFile.(type) {
	case string:
		cfgFilePath = cfgOrFile.(string)
		cfg, err = c.fetchCfg(cfgFilePath)
		if err != nil {
			return nil, nil, err
		}
	case *IConfig:
		cfg = cfgOrFile.(*IConfig)
	default:
		return nil, nil, errors.New("unknown config data")
	}

	wait := cfg.PageLoad.Wait
	selector := cfg.PageLoad.Selector
	delay := cfg.PageLoad.Sleep

	page, err := OpenPage(c.Browser, url, delay, selector, wait)
	if err != nil {
		return nil, nil, err
	}
	res, err := c.CrawlPage(page, cfg, autoDownload, closeTab)
	return res, page, err
}

func (c *Crawler) CrawlPage(page *rod.Page, cfgOrFile interface{}, autoDownload bool, closeTab bool) (*IResult, error) {
	var cfg *IConfig
	var err error
	cfgFilePath := ""
	switch cfgOrFile.(type) {
	case string:
		cfgFilePath = cfgOrFile.(string)
		cfg, err = c.fetchCfg(cfgFilePath)
		if err != nil {
			return nil, err
		}
	case *IConfig:
		cfg = cfgOrFile.(*IConfig)
	default:
		return nil, errors.New("unknown config data")
	}
	jsCode := fmt.Sprintf(`
	(cfg)=>{
		%s;
		return run(cfg);
	}`, crawlerJs)

	resultJson, err := page.Eval(jsCode, cfg)
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
				var resNode interface{}
				resNode = result.Data
				if secName != "" {
					if m, ok := resNode.(map[string]interface{}); ok {
						resNode = m[secName]
					} else {
						return nil, err
					}
				}

				if resNode != nil {
					switch resNode.(type) {
					case []interface{}:
						for _, resExtNode := range resNode.([]interface{}) {
							if extNode, oke := resExtNode.(map[string]interface{}); oke {
								c.processExtUrl(extCfg, extNode, itemName, autoDownload, closeTab)
							}
						}
					case map[string]interface{}:
						extNode := resNode.(map[string]interface{})
						c.processExtUrl(extCfg, extNode, itemName, autoDownload, closeTab)
					default:
						return nil, fmt.Errorf("unexpected externalSection type %T", resNode)
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

func (c *Crawler) processExtUrl(extCfg string, extNode map[string]interface{}, itemName string, autoDownload bool, closeTab bool) {
	extUrl := extNode[itemName].(string)
	if extUrl != "" {
		extData, _, err2 := c.CrawlUrl(extUrl, extCfg, autoDownload, closeTab)
		if err2 != nil {
			extNode[itemName] = fmt.Sprintf("an error occurred when crawling the external url: %s", extUrl)
		} else {
			extNode[itemName] = extData.Data
		}
	}
}

func (c *Crawler) AttachDefaultBrowser() *rod.Browser {
	wsURL := launcher.NewUserMode().MustLaunch()
	c.Browser = rod.New().ControlURL(wsURL).MustConnect().NoDefaultDevice()
	return c.Browser
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
	saveDir := filepath.Join(downloadRoot, subDir)
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
		fileFullPathName := filepath.Join(saveDir, dlData.FileNames[i])
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

func (c *Crawler) fetchCfg(cfgPath string) (*IConfig, error) {
	if c.CfgFetcher != nil {
		cfg, err := c.CfgFetcher(cfgPath)
		if err != nil {
			return nil, err
		} else {
			_, err2 := json.Marshal(cfg)
			return cfg, err2
		}
	} else {
		return innerFetcher(cfgPath)
	}
}

func innerFetcher(cfgFilePath string) (*IConfig, error) {
	cfgJsonStr, err := os.ReadFile(cfgFilePath)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		return nil, err
	}
	var cfg IConfig
	err = json.Unmarshal(cfgJsonStr, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func joinPath(basePath, refPath string) (string, error) {
	if strings.HasPrefix(basePath, "http://") || strings.HasPrefix(basePath, "https://") {
		base, err := url.Parse(basePath)
		if err != nil {
			return "", err
		}
		ref, err := url.Parse(refPath)
		if err != nil {
			return "", err
		}
		return base.ResolveReference(ref).String(), nil
	} else {
		if filepath.IsAbs(refPath) {
			return refPath, nil
		}
		basePath = filepath.Dir(basePath)
		p := filepath.Join(basePath, refPath)
		return filepath.Abs(p)
	}
}

//go:embed "resource/crawler.js"
var crawlerJs string
