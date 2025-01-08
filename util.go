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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
			err = WaitElementShow(page, selector, 30)
		case WaitHide:
			err = WaitElementHide(page, selector, 30)
		}
	}

	if sleep > 0 {
		time.Sleep(time.Duration(sleep) * time.Second)
	}

	return
}

// RaceShow waits for the first element to become visible from a list of selectors.
// Returns the index of the first visible element, the element itself, and any error
func RaceShow(page *rod.Page, selectors []string, timeoutSeconds int) (int, *rod.Element, error) {
	type result struct {
		index int
		elem  *rod.Element
		err   error
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	resultChan := make(chan result, 1)

	// Start concurrent checks for each selector
	for i := range selectors {
		go func(index int, selector string) {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					func() {
						defer func() {
							if r := recover(); r != nil {
								var e error
								switch x := r.(type) {
								case string:
									e = errors.New(x)
								case error:
									e = x
								default:
									e = errors.New("unknown panic")
								}
								resultChan <- result{-1, nil, e}
							}
						}()

						if ElementVisible(page, selector) {
							if elemX, errX := page.Element(selector); errX == nil {
								resultChan <- result{index, elemX, nil}
								return
							}
						}
					}()
				}
			}
		}(i, selectors[i])
	}

	select {
	case <-ctx.Done():
		return -1, nil, fmt.Errorf("timeout waiting for elements after %d seconds", timeoutSeconds)
	case r := <-resultChan:
		return r.index, r.elem, r.err
	}
}

// WaitElementHide waits for an element to become invisible on the page
func WaitElementHide(page *rod.Page, selector string, timeoutSeconds int) error {
	if !ElementVisible(page, selector) {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var lastState bool = true
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait element hide timed out after %d seconds", timeoutSeconds)
		case <-ticker.C:
			visible := ElementVisible(page, selector)
			if !lastState && !visible {
				return nil
			}
			if visible != lastState {
				lastState = false
			}
		}
	}
}

// WaitElementShow waits for an element to become visible on the page
func WaitElementShow(page *rod.Page, selector string, timeoutSeconds int) (err error) {
	if ElementVisible(page, selector) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var lastState bool
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait element show timed out after %d seconds", timeoutSeconds)
		case <-ticker.C:
			visible := ElementVisible(page, selector)
			if lastState && visible {
				return
			}
			if visible != lastState {
				lastState = true
			}
		}
	}
}

// 共享的 JavaScript 代码
const commonJSCode = `
const replacePseudo = (selector, parentElement = document) => {
    let doc = parentElement;
    const pseudoMatch = selector.match(/^:(frame|shadow)\((.+?)\)/);
    
    if (!pseudoMatch) {
        return { doc, selector, ctxChanged: false };
    }
    
    const [, pseudoType, pseudoSelector] = pseudoMatch;
    const pseudoElem = parentElement.querySelector(pseudoSelector);
    
    if (!pseudoElem) {
        return { doc, selector, ctxChanged: false };
    }
    
    doc = pseudoType === 'frame' ? pseudoElem.contentWindow.document : pseudoElem.shadowRoot;
    selector = selector.slice(pseudoMatch[0].length).trim();
    
    return /^:(frame|shadow)\(/.test(selector) ? replacePseudo(selector, doc) : { doc, selector, ctxChanged: true };
};

const queryElem = (selector, parentElement = document) => {
    const { doc, selector: finalSelector } = replacePseudo(selector, parentElement);
    return doc.querySelector(finalSelector);
};
`

// ElementVisible checks if an element is visible on the page
func ElementVisible(page *rod.Page, selector string) bool {
	const jsCode = commonJSCode + `
    (selector) => {
        try {
            const elem = queryElem(selector);
            if (!elem) return false;
            
            const { height, width } = elem.getBoundingClientRect();
            return height > 0 && width > 0;
        } catch {
            return false;
        }
    }`

	return page.MustEval(jsCode, selector).Bool()
}

// QueryElem returns the element matching the selector
func QueryElem(page *rod.Page, selector string) (*rod.Element, error) {
	const jsCode = commonJSCode + `
    (selector) => {
        try {
            return queryElem(selector) || null;
        } catch {
            return null;
        }
    }`

	opts := &rod.EvalOptions{
		JS: jsCode,
		JSArgs: []interface{}{
			selector,
		},
	}
	return page.ElementByJS(opts)
}

// RenameFileUnique generates a unique filename by appending a number if the file already exists
func RenameFileUnique(dir, fileName, ext string) string {
	for try := 0; try < 1000; try++ {
		name := fileName
		if try > 0 {
			name = fmt.Sprintf("%s_%d", fileName, try)
		}

		fullPath := filepath.Join(dir, name+ext)
		exists, err := FileExists(fullPath)
		if err != nil {
			timestamp := time.Now().UnixNano()
			return filepath.Join(dir, fmt.Sprintf("%s_%d%s", fileName, timestamp, ext))
		}

		if !exists {
			return fullPath
		}
	}

	// If we've tried 1000 times and still no success, fallback to timestamp
	timestamp := time.Now().UnixNano()
	return filepath.Join(dir, fmt.Sprintf("%s_%d%s", fileName, timestamp, ext))
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

func emptyDirectoryConcurrent(dir string, entries []os.DirEntry) error {
	const maxWorkers = 10
	numWorkers := min(maxWorkers, len(entries))

	errChan := make(chan error, len(entries))
	semaphore := make(chan struct{}, numWorkers)

	var wg sync.WaitGroup
	for _, entry := range entries {
		wg.Add(1)
		go func(e os.DirEntry) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			path := filepath.Join(dir, e.Name())
			if err := os.RemoveAll(path); err != nil {
				errChan <- fmt.Errorf("failed to remove %s: %w", path, err)
			}
		}(entry)
	}

	// Wait for all deletions to complete
	wg.Wait()
	close(errChan)

	// Check for any errors
	for err := range errChan {
		return err // Return first error encountered
	}

	return nil
}

// EmptyDirectory removes all contents of a directory while preserving the directory itself
func EmptyDirectory(dir string) error {
	if dir == "" {
		return fmt.Errorf("directory path cannot be empty")
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("failed to access directory: %w", err)
	}
	if !dirInfo.IsDir() {
		return fmt.Errorf("path %s is not a directory", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	if len(entries) == 0 {
		return nil
	}

	// Use worker pool for large directories
	if len(entries) > 100 {
		return emptyDirectoryConcurrent(dir, entries)
	}

	// Use sequential deletion for small directories
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
	}

	return nil
}

var (
	// invalidFileChars 包含 Windows 和类 Unix 系统中文件名的非法字符
	invalidFileChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)

	// multipleUnderscores 用于替换连续的下划线
	multipleUnderscores = regexp.MustCompile(`_+`)

	// reservedNames 是 Windows 系统保留的文件名
	reservedNames = map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true,
		"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true,
	}
)

// NormalizeFilename sanitizes a filename to be safe for all operating systems.
// It removes invalid characters, handles reserved names, and ensures the result
// is a valid filename.
func NormalizeFilename(name string) string {
	if name == "" {
		return "_"
	}

	// 去除首尾空格
	name = strings.TrimSpace(name)

	// 替换无效字符为下划线
	name = invalidFileChars.ReplaceAllString(name, "_")

	// 合并多个连续的下划线
	name = multipleUnderscores.ReplaceAllString(name, "_")

	// 去除首尾的下划线
	name = strings.Trim(name, "_")

	// 如果名称为空（例如，原始字符串只包含无效字符）
	if name == "" {
		return "_"
	}

	// 检查 Windows 保留名称
	upperName := strings.ToUpper(name)
	baseName := strings.Split(upperName, ".")[0]
	if reservedNames[baseName] {
		name = "_" + name
	}

	// 确保文件名不超过最大长度（Windows 限制为 255 个字符）
	const maxLength = 255
	if len(name) > maxLength {
		ext := filepath.Ext(name)
		name = name[:maxLength-len(ext)] + ext
	}

	return name
}

// GetDictAndLastSegmentByPath traverses a nested map structure using a path and returns
// the parent data, the last path segment, and any error encountered.
func GetDictAndLastSegmentByPath(data map[string]interface{}, path string) (interface{}, string, error) {
	if len(path) == 0 {
		return nil, "", fmt.Errorf("empty path")
	}
	if data == nil {
		return nil, "", fmt.Errorf("nil data")
	}

	path = strings.Trim(path, "/")
	keys := strings.Split(path, "/")
	if len(keys) == 0 {
		return nil, "", fmt.Errorf("invalid path format")
	}

	lastIndex := len(keys) - 1
	lastSegment := keys[lastIndex]

	if lastIndex == 0 {
		return data, lastSegment, nil
	}

	current := interface{}(data)
	for i, key := range keys[:lastIndex] {
		dict, ok := current.(map[string]interface{})
		if !ok {
			return nil, lastSegment, fmt.Errorf("invalid type at path segment %q",
				strings.Join(keys[:i+1], "/"))
		}

		current, ok = dict[key]
		if !ok {
			return nil, lastSegment, fmt.Errorf("key not found at path segment %q",
				strings.Join(keys[:i+1], "/"))
		}
	}

	return current, lastSegment, nil
}

type ExecuteResult struct {
	Output string
	Err    error
}

// ExecShell executes a shell command with timeout control
// ctx can be created with timeout using context.WithTimeout
func ExecShell(ctx context.Context, command string) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("context cannot be nil")
	}
	if command == "" {
		return "", fmt.Errorf("command cannot be empty")
	}

	// Create command
	cmd := exec.Command("cmd", "/C", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	// Create a buffered channel to avoid goroutine leak
	resultChan := make(chan ExecuteResult, 1)

	// Start command execution in goroutine
	go func() {
		output, err := cmd.CombinedOutput()
		resultChan <- ExecuteResult{
			Output: GBK2UTF8(string(output)),
			Err:    err,
		}
	}()

	// Wait for either command completion or context cancellation
	select {
	case <-ctx.Done():
		// Try to kill the process gracefully first
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			// If SIGTERM fails, force kill
			KillProcess(cmd.Process.Pid)
		}

		// Wait a short time for process to terminate
		time.Sleep(100 * time.Millisecond)

		// Force kill if still running
		if IsProcessRunning(cmd.Process.Pid) {
			KillProcess(cmd.Process.Pid)
		}

		return "", fmt.Errorf("command execution timeout: %w", ctx.Err())

	case result := <-resultChan:
		if result.Err != nil {
			return result.Output, fmt.Errorf("command execution failed: %w", result.Err)
		}
		return result.Output, nil
	}
}

// KillProcess forcefully terminates a process and its children
func KillProcess(pid int) {
	if pid <= 0 {
		return
	}

	killCmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
	killCmd.Run() // Ignore error as we'll try Process.Kill anyway

	if process, err := os.FindProcess(pid); err == nil {
		process.Kill() // Ignore error as process might already be dead
	}
}

// IsProcessRunning checks if a process is still running
func IsProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Windows, FindProcess always succeeds, so we need to check if we can signal it
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// GBK2UTF8 GBK编码转换为UTF8
func GBK2UTF8(s string) string {
	dec := mahonia.NewDecoder("gbk")
	if dec == nil {
		return s
	}

	return dec.ConvertString(s)
}

// ExtractUrlParam extracts a specific parameter value from a URL string
func ExtractUrlParam(urlString, paramName string) (string, error) {
	if urlString == "" || paramName == "" {
		return "", fmt.Errorf("url or parameter name cannot be empty")
	}

	parsedURL, err := url.Parse(urlString)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Parse query parameters
	query := parsedURL.Query()

	// Check if parameter exists
	if !query.Has(paramName) {
		return "", fmt.Errorf("parameter '%s' not found", paramName)
	}

	return query.Get(paramName), nil
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
