package automate

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
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
}

type locationManager []*locationProps

func update(l locationManager, e scraper.ElementLocation, s string) locationManager {
	// new implementation
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
					cc := []string{}
					for j := 0; j < len(ac); j++ {
						for k := 0; k < len(bc); k++ {
							if ac[j] == bc[k] {
								cc = append(cc, ac[j])
							}
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

func filter(l locationManager, minCount int) locationManager {
	// remove if count is smaller than minCount
	// or if the examples are all the same.
	i := 0
	for _, p := range l {
		if p.count >= minCount {
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

func elementsToConfig(s *scraper.Scraper, l ...scraper.ElementLocation) {
	var itemSelector string
outer:
	for i := 0; ; i++ {
		var c string
		for j, e := range l {
			if i >= len(selectorToPath(e.Selector)) {
				itemSelector = pathToSelector(selectorToPath(e.Selector)[:i-1])
				break outer
			}
			if j == 0 {
				c = selectorToPath(e.Selector)[i]
			} else {
				if selectorToPath(e.Selector)[i] != c {
					itemSelector = pathToSelector(selectorToPath(e.Selector)[:i])
					break outer
				}
			}
		}
	}
	s.Item = itemSelector
	for i, e := range l {
		e.Selector = strings.TrimLeft(strings.TrimPrefix(e.Selector, itemSelector), " >")
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

func GetDynamicFieldsConfig(s *scraper.Scraper, minOcc int, showDetails bool) error {
	if s.URL == "" {
		return errors.New("URL field cannot be empty")
	}
	res, err := utils.FetchUrl(s.URL, "")
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}
	z := html.NewTokenizer(res.Body)
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
				if len(strings.TrimSpace(text)) > 1 {
					l := scraper.ElementLocation{
						Selector:   p,
						ChildIndex: nrChildren[p],
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
				// what type of token is <br /> ? Same as <br> ?
				if tnString == "br" {
					nrChildren[pathToSelector(nodePath)] += 1
					continue
				}
				if tt == html.StartTagToken {
					nrChildren[pathToSelector(nodePath)] += 1
					moreAttr := true
					var hrefVal string
					for moreAttr {
						k, v, m := z.TagAttr()
						if string(k) == "class" && string(v) != "" {
							cls := strings.Split(string(v), " ")
							j := 0
							for _, cl := range cls {
								if cl != "" {
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
					if tnString != "br" {
						nodePath = append(nodePath, tnString)
						nrChildren[pathToSelector(nodePath)] = 0
						depth++
						if tnString == "a" && hrefVal != "" {
							p := pathToSelector(nodePath)
							l := scraper.ElementLocation{
								Selector:   p,
								ChildIndex: nrChildren[p],
								Attr:       "href",
							}
							locMan = update(locMan, l, hrefVal)
						}
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
		}
	}

	locMan = filter(locMan, minOcc)

	if len(locMan) > 0 {
		sort.Slice(locMan, func(p, q int) bool {
			return locMan[p].loc.Selector > locMan[q].loc.Selector
		})

		showFieldsTable(locMan, showDetails)

		reader := bufio.NewReader(os.Stdin)
		fmt.Println("please select one or more of the suggested fields by typing the according numbers separated by spaces:")
		text, _ := reader.ReadString('\n')
		var ns []int
		for _, n := range strings.Split(strings.TrimRight(text, "\n"), " ") {
			ni, err := strconv.Atoi(n)
			if err != nil {
				return fmt.Errorf("please enter valid numbers")
			}
			ns = append(ns, ni)
		}
		// ns := []int{0, 3, 4}
		var fs []scraper.ElementLocation
		for _, n := range ns {
			if n >= len(locMan) {
				return fmt.Errorf("please enter valid numbers")
			}
			fs = append(fs, locMan[n].loc)
		}

		elementsToConfig(s, fs...)
		return nil
	}
	return fmt.Errorf("no fields found")
}

func showFieldsTable(locMan locationManager, showDetails bool) {
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
				ss := utils.ShortenString(locMan[r-1].examples[c-1], 50)
				table.SetCell(r, c,
					tview.NewTableCell(ss).
						SetTextColor(color).
						SetAlign(tview.AlignCenter))
			}
		}
	}
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
		} else {
			table.GetCell(row, 0).SetTextColor(tcell.ColorGreen)
		}
	})
	if err := app.SetRoot(table, true).SetFocus(table).Run(); err != nil {
		panic(err)
	}
}
