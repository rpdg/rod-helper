package rpa

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-rod/rod"
	"os"
	"sort"
	"time"
)

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
