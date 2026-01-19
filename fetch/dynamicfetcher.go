package fetch

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/jakopako/goskyr/config"
	"github.com/jakopako/goskyr/log"
	"github.com/jakopako/goskyr/types"
	"github.com/jakopako/goskyr/utils"
)

// The DynamicFetcher renders js
type DynamicFetcher struct {
	*FetcherConfig
	allocContext context.Context
	cancelAlloc  context.CancelFunc
}

func NewDynamicFetcher(fc *FetcherConfig) *DynamicFetcher {
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.WindowSize(1920, 1080), // init with a desktop view (sometimes pages look different on mobile, eg buttons are missing)
	)
	if fc.UserAgent != "" {
		opts = append(opts,
			chromedp.UserAgent(fc.UserAgent))
	}
	allocContext, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	d := &DynamicFetcher{
		FetcherConfig: fc,
		allocContext:  allocContext,
		cancelAlloc:   cancelAlloc,
	}
	if d.PageLoadWaitMS == 0 {
		d.PageLoadWaitMS = 2000 // default
	}
	return d
}

func (d *DynamicFetcher) Cancel() {
	d.cancelAlloc()
}

func (d *DynamicFetcher) Fetch(ctx context.Context, urlStr string, opts FetchOpts) (string, error) {
	logger := log.LoggerFromContext(ctx).With(slog.String("fetcher", "dynamic"), slog.String("url", urlStr))
	logger.Debug("fetching page", slog.String("user-agent", d.UserAgent))
	// start := time.Now()
	ctx, cancel := chromedp.NewContext(d.allocContext)
	// ctx, cancel := chromedp.NewContext(d.allocContext,
	// 	chromedp.WithLogf(log.Printf),
	// 	chromedp.WithDebugf(log.Printf),
	// 	chromedp.WithErrorf(log.Printf),
	// )
	defer cancel()

	actions := []chromedp.Action{}

	// log chrome version in debug mode
	if config.Debug {
		actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
			protocolVersion, product, revision, userAgent, jsVersion, err := browser.GetVersion().Do(ctx)
			if err != nil {
				logger.Warn("failed to get chrome version", slog.String("err", err.Error()))
				return nil
			}
			logger.Debug(fmt.Sprintf("chrome version: protocolVersion=%s, product=%s, revision=%s, userAgent=%s, jsVersion=%s",
				protocolVersion, product, revision, userAgent, jsVersion))
			return nil
		}))
	}

	var body string
	sleepTime := time.Duration(d.PageLoadWaitMS) * time.Millisecond
	actions = append(actions,
		chromedp.Navigate(urlStr),
		chromedp.Sleep(sleepTime),
	)
	logger.Debug(fmt.Sprintf("appended chrome actions: Navigate, Sleep(%v)", sleepTime))
	for j, ia := range opts.Interaction {
		logger.Debug(fmt.Sprintf("processing interaction nr %d, type %s", j, ia.Type))
		delay := 500 * time.Millisecond // default is .5 seconds
		if ia.Delay > 0 {
			delay = time.Duration(ia.Delay) * time.Millisecond
		}
		switch ia.Type {
		case types.InteractionTypeClick:
			count := 1 // default is 1
			if ia.Count > 0 {
				count = ia.Count
			}
			for range count {
				// we only click the button if it exists. Do we really need this check here?
				actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
					var nodes []*cdp.Node
					if err := chromedp.Nodes(ia.Selector, &nodes, chromedp.AtLeast(0)).Do(ctx); err != nil {
						return err
					}
					if len(nodes) == 0 {
						return nil
					} // nothing to do
					logger.Debug(fmt.Sprintf("clicking on node with selector: %s", ia.Selector))
					return chromedp.MouseClickNode(nodes[0]).Do(ctx)
				}))
				actions = append(actions, chromedp.Sleep(delay))
				logger.Debug(fmt.Sprintf("appended chrome actions: ActionFunc (mouse click), Sleep(%v)", delay))
			}
		case types.InteractionTypeScroll:
			// scroll to the bottom of the page
			actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
				logger.Debug("scrolling down the page")
				return chromedp.KeyEvent(kb.End).Do(ctx)
			}))
			actions = append(actions, chromedp.Sleep(delay))
			logger.Debug(fmt.Sprintf("appended chrome actions: ActionFunc (scroll down), Sleep(%v)", delay))
		default:
			logger.Warn(fmt.Sprintf("unknown interaction type %s", ia.Type))
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

	if config.Debug {
		// ensure debug directory exists
		if d.DebugDir != "" {
			err := os.MkdirAll(d.DebugDir, os.ModePerm)
			if err != nil {
				return "", fmt.Errorf("failed to create debug directory: %v", err)
			}
		}

		u, _ := url.Parse(urlStr)
		var buf []byte
		r, err := utils.RandomString(u.Host)
		if err != nil {
			return "", err
		}
		filename := path.Join(d.DebugDir, fmt.Sprintf("%s.png", r))
		actions = append(actions, chromedp.CaptureScreenshot(&buf))
		actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
			logger.Debug(fmt.Sprintf("writing screenshot to file %s", filename))
			return os.WriteFile(filename, buf, 0644)
		}))
		logger.Debug("appended chrome actions: CaptureScreenshot, ActionFunc (save screenshot)")
	}

	// run task list
	err := chromedp.Run(ctx,
		actions...,
	)
	// elapsed := time.Since(start)
	// log.Printf("fetching %s took %s", url, elapsed)

	if err != nil {
		return "", err
	}

	if config.Debug {
		writeHTMLToFile(ctx, urlStr, body, d.DebugDir)
	}
	return body, nil
}
