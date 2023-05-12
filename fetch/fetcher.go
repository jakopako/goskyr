package fetch

import (
	"context"
	"fmt"
	"io"
	"log"
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
	UserAgent   string
	Interaction Interaction
	WaitSeconds int
}

type Interaction struct {
	Selector string
	Count    int
}

func (d *DynamicFetcher) Fetch(url string) (string, error) {
	// TODO: add user agent
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(1920, 1080), // init with a desktop view (sometimes pages look different on mobile, eg buttons are missing)
	)
	parentCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	ctx, cancel := chromedp.NewContext(parentCtx, chromedp.WithDebugf(log.Printf))
	defer cancel()

	var body string
	sleepTime := 5 * time.Second
	if d.WaitSeconds > 0 {
		sleepTime = time.Duration(d.WaitSeconds) * time.Second
	}
	actions := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.Sleep(sleepTime), // for now
	}
	if d.Interaction.Selector != "" {
		count := 1
		if d.Interaction.Count > 0 {
			count = d.Interaction.Count
		}
		for i := 0; i < count; i++ {
			actions = append(actions, chromedp.Click(d.Interaction.Selector, chromedp.ByQuery))
			actions = append(actions, chromedp.Sleep(sleepTime))
		}
	}
	actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
		node, err := dom.GetDocument().Do(ctx)
		if err != nil {
			return err
		}
		body, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
		return err
	}))

	// run task list
	err := chromedp.Run(ctx,
		actions...,
	)
	return body, err
}
