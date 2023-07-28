package rpa

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
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

type PageLoad struct {
	Wait     WaitSign `json:"wait"`
	Selector string   `json:"selector,omitempty"`
	Sleep    int64    `json:"sleep,omitempty"`
}

type ConfigNode struct {
	Selector string `json:"selector"`
	Label    string `json:"label"`
	ID       string `json:"id"`
}

type DownloadType string

const (
	DownloadUrl     DownloadType = "url"
	DownloadElement DownloadType = "element"
)

type DownloadConfig struct {
	ConfigNode
	SavePath   string       `json:"savePath,omitempty"`
	NameProper string       `json:"nameProper,omitempty"`
	NameRender string       `json:"nameRender,omitempty"`
	Type       DownloadType `json:"type"`
}

type CrawlerConfig struct {
	PageLoad        PageLoad                 `json:"pageLoad,omitempty"`
	DataSection     []map[string]interface{} `json:"dataSection"`
	SwitchSection   map[string]interface{}   `json:"switchSection,omitempty"`
	DownloadRoot    string                   `json:"downloadRoot,omitempty"`
	DownloadSection []DownloadConfig         `json:"downloadSection,omitempty"`
}

// DownloadResult is a part of result section
type DownloadResult struct {
	Count     int      `json:"count"`
	Errors    []int    `json:"errors"`
	FileNames []string `json:"fileNames"`
	Links     []string `json:"links"`
}

type ExternalResult struct {
	Config  string `json:"config"`
	Connect string `json:"connect"`
	ID      string `json:"id"`
}

type Result struct {
	Data            map[string]interface{}    `json:"data"`
	DownloadRoot    string                    `json:"downloadRoot"`
	Downloads       map[string]DownloadResult `json:"downloads"`
	ExternalSection map[string]ExternalResult `json:"externalSection"`
}

type Crawler struct {
	Browser    *rod.Browser
	CfgFetcher func(path string) (*CrawlerConfig, error)
}

func (c *Crawler) CrawlUrl(url string, cfgOrFile interface{}, autoDownload bool, closeTab bool) (*Result, *rod.Page, error) {
	var err error

	page, err := c.Browser.Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return nil, nil, err
	}

	res, err := c.CrawlPage(page, cfgOrFile, autoDownload, closeTab)
	return res, page, err
}

func (c *Crawler) CrawlPage(page *rod.Page, cfgOrFile interface{}, autoDownload bool, closeTab bool) (*Result, error) {
	var cfg *CrawlerConfig
	var err error

	cfgFilePath := ""
	switch cfgOrFile.(type) {
	case string:
		cfgFilePath = cfgOrFile.(string)
		cfg, err = c.fetchCfg(cfgFilePath)
		if err != nil {
			return nil, err
		}
	case *CrawlerConfig:
		cfg = cfgOrFile.(*CrawlerConfig)
	default:
		return nil, errors.New("unknown config data")
	}

	wait := cfg.PageLoad.Wait
	selector := cfg.PageLoad.Selector
	delay := cfg.PageLoad.Sleep
	err = WaitPage(page, delay, selector, wait)
	if err != nil {
		return nil, err
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

	var result Result
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
			extNode[itemName] = fmt.Sprintf("an error occurred when crawling the external url: %s", err2)
		} else {
			extNode[itemName] = extData.Data
		}
	}
}

func (c *Crawler) AttachDefaultBrowser() error {
	br, err := ConnectDefaultBrowser(true, false)
	if err != nil {
		return err
	}
	c.Browser = br
	return nil
}

func (c *Crawler) AttachChromeBrowser() error {
	br, err := ConnectChromeBrowser(true, false)
	if err != nil {
		return err
	}
	c.Browser = br
	return nil
}

func (c *Crawler) AttachEdgeBrowser(ieMode bool) error {
	br, err := ConnectEdgeBrowser(true, false, ieMode)
	if err != nil {
		return err
	}
	c.Browser = br
	return nil
}

func (c *Crawler) download(page *rod.Page, dlCfg DownloadConfig, dlData *DownloadResult, downloadRoot string) error {
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

func (c *Crawler) fetchCfg(cfgPath string) (*CrawlerConfig, error) {
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

func innerFetcher(cfgFilePath string) (*CrawlerConfig, error) {
	cfgJsonStr, err := os.ReadFile(cfgFilePath)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		return nil, err
	}
	var cfg CrawlerConfig
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
