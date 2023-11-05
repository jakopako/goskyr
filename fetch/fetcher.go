package fetch

import (
	"context"
	"fmt"
	"io"
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
	UserAgent string
	// Interaction types.Interaction
	WaitSeconds int
	ctx         context.Context
}

func NewDynamicFetcher(ua string, s int) *DynamicFetcher {
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(1920, 1080), // init with a desktop view (sometimes pages look different on mobile, eg buttons are missing)
	)
	parentCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, _ := chromedp.NewContext(parentCtx)
	// TODO don't forget to actually do something with the context.CancelFunc
	return &DynamicFetcher{
		UserAgent:   ua,
		WaitSeconds: s,
		ctx:         ctx,
	}
}

func (d *DynamicFetcher) Fetch(url string, opts FetchOpts) (string, error) {
	// TODO: add user agent
	start := time.Now()
	// opts := append(
	// 	chromedp.DefaultExecAllocatorOptions[:],
	// 	chromedp.WindowSize(1920, 1080), // init with a desktop view (sometimes pages look different on mobile, eg buttons are missing)
	// )
	// parentCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	// elapsed := time.Since(start)
	// fmt.Printf("time elapsed: %s\n", elapsed)
	// defer cancel()
	// ctx, cancel := chromedp.NewContext(parentCtx)
	// elapsed = time.Since(start)
	// fmt.Printf("time elapsed: %s\n", elapsed)
	// ctx, cancel := chromedp.NewContext(parentCtx, chromedp.WithDebugf(log.Printf))
	// defer cancel()

	var body string
	sleepTime := 2 * time.Second
	if d.WaitSeconds > 0 {
		sleepTime = time.Duration(d.WaitSeconds) * time.Second
	}
	actions := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.Sleep(sleepTime), // for now
	}
	delay := 500 * time.Millisecond // default is .5 seconds
	if opts.Interaction.Delay > 0 {
		delay = time.Duration(opts.Interaction.Delay) * time.Millisecond
	}
	if opts.Interaction.Type == types.InteractionTypeClick {
		count := 1 // default is 1
		fmt.Println("shouldnt get here")
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

	elapsed := time.Since(start)
	fmt.Printf("time elapsed: %s\n", elapsed)
	// run task list
	err := chromedp.Run(d.ctx,
		actions...,
	)
	elapsed = time.Since(start)
	fmt.Printf("time elapsed: %s\n", elapsed)
	return body, err
}
