
# Overview


[![Go Reference](https://pkg.go.dev/badge/github.com/rpdg/rod-helper.svg)](https://pkg.go.dev/github.com/rpdg/rod-helper)


A library of auxiliary tools for [rod](https://github.com/go-rod/rod), the goal is to simplify the process of rod scraping web data through configurability. The final output can be changed by modifying the configuration file without having to recompile the program.

# Usage

```go
func main() {
	r := rpa.Crawler{}
	r.AttachDefaultBrowser()
	b := r.Browser
	b.Close()

	url := "https://cn.bing.com/search?q=sample+simple+pdf"
	val, _, err := r.CrawlUrl(url, "./sample/bing.json", true, true)
	if err != nil {
		fmt.Println(err)
	} else {
		s, _ := json.MarshalIndent(val, "", "\t")
		fmt.Println(string(s))
	}
}
```



# Custom Pseudo class

1. select element under iframe / frame:

    ```css
    :frame( iframe_tag_selector ) inner_element_selector
    ```

2. select element under shadow-dom

	```css
	:shadow(web_component_selector) inner_element_selector
	```
	
	
