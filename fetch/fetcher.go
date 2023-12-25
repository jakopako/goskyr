package fetch

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"github.com/jakopako/goskyr/types"
)

type FetchOpts struct {
	Interaction types.Interaction
}

// A Fetcher allows to fetch the content of a web page
type Fetcher interface {
	Fetch(url string, opts FetchOpts) (string, error)
}

// The StaticFetcher fetches static page content
type StaticFetcher struct {
	UserAgent string
}

func (s *StaticFetcher) Fetch(url string, opts FetchOpts) (string, error) {
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
	UserAgent        string
	WaitMilliseconds int
	allocContext     context.Context
	cancelAlloc      context.CancelFunc
}

func NewDynamicFetcher(ua string, ms int) *DynamicFetcher {
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(1920, 1080), // init with a desktop view (sometimes pages look different on mobile, eg buttons are missing)
	)
	if ua != "" {
		opts = append(opts,
			chromedp.UserAgent(ua))
	}
	allocContext, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	d := &DynamicFetcher{
		UserAgent:        ua,
		WaitMilliseconds: ms,
		allocContext:     allocContext,
		cancelAlloc:      cancelAlloc,
	}
	if d.WaitMilliseconds == 0 {
		d.WaitMilliseconds = 2000 // default
	}
	return d
}

func (d *DynamicFetcher) Cancel() {
	d.cancelAlloc()
}

func (d *DynamicFetcher) Fetch(url string, opts FetchOpts) (string, error) {
	start := time.Now()
	ctx, cancel := chromedp.NewContext(d.allocContext)
	// ctx, cancel := chromedp.NewContext(d.allocContext,
	// 	chromedp.WithLogf(log.Printf),
	// 	chromedp.WithDebugf(log.Printf),
	// 	chromedp.WithErrorf(log.Printf),
	// )
	defer cancel()
	var body string
	sleepTime := time.Duration(d.WaitMilliseconds) * time.Millisecond
	actions := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.Sleep(sleepTime),
	}
	delay := 500 * time.Millisecond // default is .5 seconds
	if opts.Interaction.Delay > 0 {
		delay = time.Duration(opts.Interaction.Delay) * time.Millisecond
	}
	if opts.Interaction.Type == types.InteractionTypeClick {
		count := 1 // default is 1
		if opts.Interaction.Count > 0 {
			count = opts.Interaction.Count
		}
		for i := 0; i < count; i++ {
			// we only click the button if it exists. Do we really need this check here?
			// TODO: should we click as many times as possible if count == 0? How would we implement this?
			// actions = append(actions, chromedp.Click(d.Interaction.Selector, chromedp.ByQuery))
			actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
				var nodes []*cdp.Node
				if err := chromedp.Nodes(opts.Interaction.Selector, &nodes, chromedp.AtLeast(0)).Do(ctx); err != nil {
					return err
				}
				if len(nodes) == 0 {
					return nil
				} // nothing to do
				return chromedp.MouseClickNode(nodes[0]).Do(ctx)
			}))
			actions = append(actions, chromedp.Sleep(delay))
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
	elapsed := time.Since(start)
	log.Printf("fetching %s took %s", url, elapsed)
	return body, err
}
