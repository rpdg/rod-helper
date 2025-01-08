package rpa

import (
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"testing"
)

func getPage() (*rod.Page, error) {
	b, e := ConnectChromiumBrowser(true, false)
	if e != nil {
		return nil, e
	}

	p, e := b.Page(proto.TargetCreateTarget{
		URL:       `https://goplay.tools/`,
		NewWindow: false,
	})
	if e != nil {
		return nil, e
	}

	return p, nil
}

func Test_WaitElementHide(t *testing.T) {
	p, e := getPage()
	if e != nil {
		t.Fatal(e)
	}

	e = WaitElementShow(p, ".app-preloader__content", 20)
	if e != nil {
		t.Fatal(e)
	}
	e = WaitElementHide(p, ".app-preloader__content", 20)
	if e != nil {
		t.Fatal(e)
	}
	t.Log("loading hidden")

	e = WaitElementShow(p, ".view-lines", 20)
	if e != nil {
		t.Fatal(e)
	}
	t.Log("editor shown")
}

func Test_QueryElem(t *testing.T) {
	p, err := getPage()
	if err != nil {
		t.Fatal(err)
	}

	err = WaitElementShow(p, ".header__logo", 20)
	if err != nil {
		t.Fatal(err)
	}

	ele, err := QueryElem(p, ".header__logo")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ele)
}

func Test_RaceShow(t *testing.T) {
	p, err := getPage()
	if err != nil {
		t.Fatal(err)
	}

	idx, ele, err := RaceShow(p, []string{".header__logo", ".app-preloader__content", ".view-lines"}, 20)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(idx, ele)
}
