//go:build windows

package resource

import (
	_ "embed"
)

//go:embed crawler.js
var CrawlerJs []byte
