package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
)

// A Fetcher allows to fetch the content of a web page
type Fetcher interface {
	Fetch(url string) (string, error)
}

// The StaticFetcher fetches static page content
type StaticFetcher struct {
	UserAgent string
}

func (s *StaticFetcher) Fetch(url string) (string, error) {
	var resString string
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return resString, err
	}
	req.Header.Set("User-Agent", s.UserAgent)
	req.Header.Set("Accept", "*/*")
	res, err := client.Do(req)
	if err != nil {
		return resString, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return resString, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}
	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return resString, err
	}
	resString = string(bytes)
	return resString, nil
}

// The DynamicFetcher renders js
type DynamicFetcher struct {
	UserAgent string
}

func (d *DynamicFetcher) Fetch(url string) (string, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// TODO: add user agent

	// run task list
	var body string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(5*time.Second), // for now
		chromedp.ActionFunc(func(ctx context.Context) error {
			node, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}
			body, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
			return err
		}),
	)
	return body, err
}
