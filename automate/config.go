package automate

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/agnivade/levenshtein"
	"github.com/gdamore/tcell/v2"
	"github.com/jakopako/goskyr/fetch"
	"github.com/jakopako/goskyr/scraper"
	"github.com/jakopako/goskyr/utils"
	"github.com/rivo/tview"
	"golang.org/x/net/html"
)

type locationProps struct {
	loc      scraper.ElementLocation
	count    int
	examples []string
	selected bool
	color    tcell.Color
	distance float64
}

type locationManager []*locationProps

func (l locationManager) setColors() {
	if len(l) == 0 {
		return
	}
	for i, e := range l {
		if i != 0 {
			e.distance = l[i-1].distance + distance(l[i-1].loc, e.loc)
		}
	}
	// scale to 1 and map to rgb
	maxDist := l[len(l)-1].distance * 1.2
	s := 0.73
	v := 0.96
	for _, e := range l {
		h := e.distance / maxDist
		r, g, b := utils.HSVToRGB(h, s, v)
		e.color = tcell.NewRGBColor(r, g, b)
	}
}

func distance(loc1, loc2 scraper.ElementLocation) float64 {
	// calculate differently? eg with nodes of html tree. eg nodes to walk to get from loc1 to loc2
	return float64(levenshtein.ComputeDistance(loc1.Selector, loc2.Selector))
}

func update(l locationManager, e scraper.ElementLocation, s string) locationManager {
	for _, lp := range l {
		if checkAndUpdatePath(&lp.loc, &e) {
			lp.count++
			if lp.count <= 4 {
				lp.examples = append(lp.examples, s)
			}
			return l
		}
	}
	return append(l, &locationProps{loc: e, count: 1, examples: []string{s}})
}

func checkAndUpdatePath(a, b *scraper.ElementLocation) bool {
	// returns true if the paths overlap and the rest of the
	// element location is identical. If true is returned
	// the Selector of a will be updated if necessary.
	if a.NodeIndex == b.NodeIndex && a.ChildIndex == b.ChildIndex && a.Attr == b.Attr {
		if a.Selector == b.Selector {
			return true
		} else {
			ap := selectorToPath(a.Selector)
			bp := selectorToPath(b.Selector)
			np := []string{}
			if len(ap) != len(bp) {
				return false
			}
			for i, an := range ap {
				ae, be := strings.Split(an, "."), strings.Split(bp[i], ".")
				at, bt := ae[0], be[0]
				if at == bt {
					if len(ae) == 1 && len(be) == 1 {
						np = append(np, an)
						continue
					}
					ac, bc := ae[1:], be[1:]
					sort.Strings(ac)
					sort.Strings(bc)

					cc := []string{}
					// find overlapping classes
					for j, k := 0, 0; j < len(ac) && k < len(bc); {
						if ac[j] == bc[k] {
							cc = append(cc, ac[j])
							j++
							k++
						} else if ac[j] > bc[k] {
							k++
						} else {
							j++
						}
					}

					if len(cc) > 0 {
						nnl := append([]string{at}, cc...)
						nn := strings.Join(nnl, ".")
						np = append(np, nn)
						continue
					}

				}
				return false

			}
			// if we get until here there is an overlapping path
			a.Selector = pathToSelector(np)
			return true
		}
	}
	return false
}

func filter(l locationManager, minCount int, removeStaticFields bool) locationManager {
	// remove if count is smaller than minCount
	// or if the examples are all the same (if removeStaticFields is true)
	i := 0
	for _, p := range l {
		if p.count >= minCount {
			if removeStaticFields {
				eqEx := true
				for _, ex := range p.examples {
					if ex != p.examples[0] {
						eqEx = false
						break
					}
				}
				if !eqEx {
					l[i] = p
					i++
				}
			} else {
				l[i] = p
				i++
			}
		}
	}
	return l[:i]
}

func pathToSelector(pathSlice []string) string {
	return strings.Join(pathSlice, " > ")
}

func selectorToPath(s string) []string {
	return strings.Split(s, " > ")
}

func nodesEqual(n1, n2 string) bool {
	if n1 == n2 {
		return true
	}
	nl1, nl2 := strings.Split(n1, "."), strings.Split(n2, ".")
	if nl1[0] == nl2[0] {
		lnl1, lnl2 := len(nl1), len(nl2)
		if lnl1 == lnl2 {
			if lnl1 > 1 {
				cn1, cn2 := nl1[1:], nl2[1:]
				sort.Strings(cn1)
				sort.Strings(cn2)
				for i := 0; i < len(cn1); i++ {
					if cn1[i] != cn2[i] {
						return false
					}
				}
				return true
			}
		}
	}
	return false
}

func removeNodesPrefix(s1 string, n int) string {
	return pathToSelector(selectorToPath(s1)[n:])
}

func escapeCssSelector(s string) string {
	return escapeNumber(escapeColons(s))
}

func escapeColons(s string) string {
	// https://www.itsupportguides.com/knowledge-base/website-tips/css-colon-in-id/
	return strings.ReplaceAll(s, ":", "\\:")
}

func escapeNumber(s string) string {
	// https://stackoverflow.com/questions/45293534/css-class-starting-with-number-is-not-getting-applied
	e := ""
	sr := []rune(s)
	for i, c := range s {
		if unicode.IsDigit(c) && string(sr[i-1]) == "." {
			e += fmt.Sprintf(`\3%s `, string(c))
		} else {
			e += string(c)
		}
	}
	return e
}

func elementsToConfig(s *scraper.Scraper, l ...scraper.ElementLocation) {
	var itemSelector string
outer:
	for i := 0; ; i++ {
		var n string
		for j, e := range l {
			if i >= len(selectorToPath(e.Selector)) {
				itemSelector = pathToSelector(selectorToPath(e.Selector)[:i])
				break outer
			}
			if j == 0 {
				n = selectorToPath(e.Selector)[i]
			} else {
				if !nodesEqual(selectorToPath(e.Selector)[i], n) {
					itemSelector = pathToSelector(selectorToPath(e.Selector)[:i])
					break outer
				}
			}
		}
	}
	s.Item = escapeCssSelector(itemSelector)
	for i, e := range l {
		e.Selector = removeNodesPrefix(e.Selector, len(strings.Split(itemSelector, " > ")))
		e.Selector = escapeCssSelector(e.Selector)
		fieldType := "text"
		if e.Attr == "href" {
			fieldType = "url"
		}
		d := scraper.Field{
			Name:            fmt.Sprintf("field-%d", i),
			Type:            fieldType,
			ElementLocation: e,
		}
		s.Fields = append(s.Fields, d)
	}
}

func GetDynamicFieldsConfig(s *scraper.Scraper, minOcc int, removeStaticFields bool) error {
	if s.URL == "" {
		return errors.New("URL field cannot be empty")
	}
	s.Name = s.URL

	var fetcher fetch.Fetcher
	if s.RenderJs {
		fetcher = &fetch.DynamicFetcher{}
	} else {
		fetcher = &fetch.StaticFetcher{}
	}
	res, err := fetcher.Fetch(s.URL)
	if err != nil {
		return err
	}

	// A bit hacky. But goquery seems to manipulate the html (I only know of goquery adding tbody tags if missing)
	// so we rely on goquery to read the html for both scraping AND figuring out the scraping config.
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(res))
	if err != nil {
		return err
	}

	// Now we have to translate the goquery doc back into a string
	htmlStr, err := goquery.OuterHtml(doc.Children())
	if err != nil {
		return err
	}
	z := html.NewTokenizer(strings.NewReader(htmlStr))
	locMan := locationManager{}
	nrChildren := map[string]int{}
	nodePath := []string{}
	depth := 0
	inBody := false
parse:
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			break parse
		case html.TextToken:
			if inBody {
				text := string(z.Text())
				p := pathToSelector(nodePath)
				if len(strings.TrimSpace(text)) > 0 {
					cI := nrChildren[p]
					l := scraper.ElementLocation{
						Selector:   p,
						ChildIndex: cI,
					}
					locMan = update(locMan, l, strings.TrimSpace(text))
				}
				nrChildren[p] += 1
			}
		case html.StartTagToken, html.EndTagToken:
			tn, _ := z.TagName()
			tnString := string(tn)
			if tnString == "body" {
				inBody = !inBody
			}
			if inBody {
				// br can also be self closing tag, see later case statement
				if tnString == "br" || tnString == "input" {
					nrChildren[pathToSelector(nodePath)] += 1
					continue
				}
				if tt == html.StartTagToken {
					nrChildren[pathToSelector(nodePath)] += 1
					moreAttr := true
					var hrefVal string
					for moreAttr {
						k, v, m := z.TagAttr()
						vString := strings.TrimSpace(string(v))
						if string(k) == "class" && vString != "" {
							cls := strings.Split(vString, " ")
							j := 0
							for _, cl := range cls {
								// for now we ignore classes that contain dots
								if cl != "" && !strings.Contains(cl, ".") {
									cls[j] = cl
									j++
								}
							}
							cls = cls[:j]
							tnString += fmt.Sprintf(".%s", strings.Join(cls, "."))
						}
						if string(k) == "href" {
							hrefVal = string(v)
						}
						moreAttr = m
					}
					nodePath = append(nodePath, tnString)
					nrChildren[pathToSelector(nodePath)] = 0
					depth++
					if (strings.HasPrefix(tnString, "a.") || tnString == "a") && hrefVal != "" {
						p := pathToSelector(nodePath)
						l := scraper.ElementLocation{
							Selector:   p,
							ChildIndex: nrChildren[p],
							Attr:       "href",
						}
						locMan = update(locMan, l, hrefVal)
					}
				} else {
					n := true
					for n && depth > 0 {
						if strings.Split(nodePath[len(nodePath)-1], ".")[0] == tnString {
							if tnString == "body" {
								break parse
							}
							n = false
						}
						delete(nrChildren, pathToSelector(nodePath))
						nodePath = nodePath[:len(nodePath)-1]
						depth--
					}
				}
			}
		case html.SelfClosingTagToken:
			if inBody {
				tn, _ := z.TagName()
				tnString := string(tn)
				if tnString == "br" || tnString == "input" {
					nrChildren[pathToSelector(nodePath)] += 1
					continue
				}
			}
		}
	}

	locMan = filter(locMan, minOcc, removeStaticFields)
	locMan.setColors()

	if len(locMan) > 0 {
		selectFieldsTable(locMan)

		var fs []scraper.ElementLocation
		for _, lm := range locMan {
			if lm.selected {
				fs = append(fs, lm.loc)
			}
		}

		if len(fs) > 0 {
			elementsToConfig(s, fs...)
			return nil
		}
		return fmt.Errorf("no fields selected")
	}
	return fmt.Errorf("no fields found")
}

func selectFieldsTable(locMan locationManager) {
	app := tview.NewApplication()
	table := tview.NewTable().SetBorders(true)
	cols, rows := 5, len(locMan)+1
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			color := tcell.ColorWhite
			if c < 1 || r < 1 {
				if c < 1 && r > 0 {
					color = tcell.ColorGreen
					table.SetCell(r, c, tview.NewTableCell(fmt.Sprintf("field [%d]", r-1)).
						SetTextColor(color).
						SetAlign(tview.AlignCenter))
				} else if r == 0 && c > 0 {
					color = tcell.ColorBlue
					table.SetCell(r, c, tview.NewTableCell(fmt.Sprintf("example [%d]", c-1)).
						SetTextColor(color).
						SetAlign(tview.AlignCenter))
				} else {
					table.SetCell(r, c,
						tview.NewTableCell("").
							SetTextColor(color).
							SetAlign(tview.AlignCenter))
				}
			} else {
				var ss string
				if len(locMan[r-1].examples) >= c {
					ss = utils.ShortenString(locMan[r-1].examples[c-1], 40)
				}
				table.SetCell(r, c,
					tview.NewTableCell(ss).
						SetTextColor(locMan[r-1].color).
						SetAlign(tview.AlignCenter))
			}
		}
	}
	table.SetSelectable(true, false)
	table.Select(1, 1).SetFixed(1, 1).SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			app.Stop()
		}
		if key == tcell.KeyEnter {
			table.SetSelectable(true, false)
		}
	}).SetSelectedFunc(func(row int, column int) {
		locMan[row-1].selected = !locMan[row-1].selected
		if locMan[row-1].selected {
			table.GetCell(row, 0).SetTextColor(tcell.ColorRed)
			for i := 1; i < 5; i++ {
				table.GetCell(row, i).SetTextColor(tcell.ColorOrange)
			}
		} else {
			table.GetCell(row, 0).SetTextColor(tcell.ColorGreen)
			for i := 1; i < 5; i++ {
				table.GetCell(row, i).SetTextColor(locMan[row-1].color)
			}
		}
	})
	button := tview.NewButton("Hit Enter to generate config").SetSelectedFunc(func() {
		app.Stop()
	})

	grid := tview.NewGrid().SetRows(-11, -1).SetColumns(-1, -1, -1).SetBorders(false).
		AddItem(table, 0, 0, 1, 3, 0, 0, true).
		AddItem(button, 1, 1, 1, 1, 0, 0, false)
	grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			if button.HasFocus() {
				app.SetFocus(table)
			} else {
				app.SetFocus(button)
			}
			return nil
		}
		return event
	})

	if err := app.SetRoot(grid, true).SetFocus(grid).Run(); err != nil {
		panic(err)
	}
}
