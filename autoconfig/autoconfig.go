package autoconfig

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gdamore/tcell/v2"
	"github.com/jakopako/goskyr/date"
	"github.com/jakopako/goskyr/fetch"
	"github.com/jakopako/goskyr/ml"
	"github.com/jakopako/goskyr/scraper"
	"github.com/jakopako/goskyr/utils"
	"github.com/rivo/tview"
)

func (l fieldManager) setColors() {
	if len(l) == 0 {
		return
	}
	for i, e := range l {
		if i != 0 {
			e.distance = l[i-1].distance + l[i-1].path.distance(e.path)
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

func (l fieldManager) findFieldNames(modelName, wordsDir string) error {
	if modelName != "" {
		ll, err := ml.LoadLabler(modelName, wordsDir)
		if err != nil {
			return err
		}
		for _, e := range l {
			pred, err := ll.PredictLabel(e.examples...)
			if err != nil {
				return err
			}
			e.name = pred // TODO: if label has occured already, add index (eg text-1, text-2...)
		}
	} else {
		for i, e := range l {
			e.name = fmt.Sprintf("field-%d", i)
		}
	}
	return nil
}

func (l fieldManager) selectFieldsTable() {
	app := tview.NewApplication()
	table := tview.NewTable().SetBorders(true)
	cols, rows := 5, len(l)+1
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			color := tcell.ColorWhite
			if c < 1 || r < 1 {
				if c < 1 && r > 0 {
					color = tcell.ColorGreen
					table.SetCell(r, c, tview.NewTableCell(fmt.Sprintf("[%d] %s", r-1, l[r-1].name)).
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
				if len(l[r-1].examples) >= c {
					ss = utils.ShortenString(l[r-1].examples[c-1], 40)
				}
				table.SetCell(r, c,
					tview.NewTableCell(ss).
						SetTextColor(l[r-1].color).
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
		l[row-1].selected = !l[row-1].selected
		if l[row-1].selected {
			table.GetCell(row, 0).SetTextColor(tcell.ColorRed)
			for i := 1; i < 5; i++ {
				table.GetCell(row, i).SetTextColor(tcell.ColorOrange)
			}
		} else {
			table.GetCell(row, 0).SetTextColor(tcell.ColorGreen)
			for i := 1; i < 5; i++ {
				table.GetCell(row, i).SetTextColor(l[row-1].color)
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

func (l fieldManager) elementsToConfig(s *scraper.Scraper) error {
	var locPropsSel []*fieldProps
	for _, lm := range l {
		if lm.selected {
			locPropsSel = append(locPropsSel, lm)
		}
	}
	if len(locPropsSel) == 0 {
		return fmt.Errorf("no fields selected")
	}

	// find shared root selector
	var rootSelector path
outer:
	for i := 0; ; i++ {
		var n node
		for j, e := range locPropsSel {
			if i >= len(e.path) {
				rootSelector = e.path[:i]
				break outer
			}
			if j == 0 {
				n = e.path[i]
			} else {
				if !n.equals(e.path[i]) {
					rootSelector = e.path[:i]
					break outer
				}
			}
		}
	}
	s.Item = shortenRootSelector(rootSelector).string()
	// for now we assume that there will only be one date field
	t := time.Now()
	zone, _ := t.Zone()
	zone = strings.Replace(zone, "CEST", "CET", 1) // quick hack for issue #209
	dateField := scraper.Field{
		Name:         "date",
		Type:         "date",
		DateLocation: zone,
	}
	for _, e := range locPropsSel {
		loc := scraper.ElementLocation{
			Selector:   e.path[len(rootSelector):].string(),
			ChildIndex: e.textIndex,
			Attr:       e.attr,
		}
		fieldType := "text"
		var d scraper.Field
		if strings.HasPrefix(e.name, "date-component") {
			cd := date.CoveredDateParts{
				Day:   strings.Contains(e.name, "day"),
				Month: strings.Contains(e.name, "month"),
				Year:  strings.Contains(e.name, "year"),
				Time:  strings.Contains(e.name, "time"),
			}
			format, lang := date.GetDateFormatMulti(e.examples, cd)
			dateField.Components = append(dateField.Components, scraper.DateComponent{
				ElementLocation: loc,
				Covers:          cd,
				Layout:          []string{format},
			})
			if dateField.DateLanguage == "" {
				// first lang wins
				dateField.DateLanguage = lang
			}
		} else {
			if loc.Attr == "href" || loc.Attr == "src" {
				fieldType = "url"
			}
			d = scraper.Field{
				Name:             e.name,
				Type:             fieldType,
				ElementLocations: []scraper.ElementLocation{loc},
			}
			s.Fields = append(s.Fields, d)
		}
	}
	if len(dateField.Components) > 0 {
		s.Fields = append(s.Fields, dateField)
	}
	return nil
}

func shortenRootSelector(p path) path {
	// the following algorithm is a bit arbitrary. Let's
	// see if it works.
	nrTotalClasses := 0
	thresholdTotalClasses := 3
	for i := len(p) - 1; i >= 0; i-- {
		nrTotalClasses += len(p[i].classes)
		if nrTotalClasses >= thresholdTotalClasses {
			return p[i:]
		}
	}
	return p
}

// squashFieldManager merges different locationProps into one
// based on their similarity. The tricky question is 'when are two
// locationProps close enough to be merged into one?'
func squashFieldManager(l fieldManager, minOcc int) fieldManager {
	squashed := fieldManager{}
	for i := len(l) - 1; i >= 0; i-- {
		lp := l[i]
		updated := false
		for _, sp := range squashed {
			updated = checkAndUpdateLocProps(sp, lp)
			if updated {
				break
			}
		}
		if !updated {
			stripNthChild(lp, minOcc)
			squashed = append(squashed, lp)
		}
	}
	return squashed
}

// stripNthChild tries to find the index in a locationProps path under which
// we need to strip the nth-child pseudo class. We need to strip that pseudo
// class because at a later point we want to find a common base path between
// different paths but if all paths' base paths look differently (because their
// nodes have different nth-child pseudo classes) there won't be a common
// base path.
func stripNthChild(lp *fieldProps, minOcc int) {
	iStrip := 0
	// every node in lp.path with index < than iStrip needs no be stripped
	// of its pseudo classes. iStrip changes during the execution of
	// this function.
	// A bit arbitrary (and probably not always correct) but
	// for now we assume that iStrip cannot be len(lp.path)-1
	// not correct for https://huxleysneuewelt.com/shows
	// but needed for http://www.bar-laparenthese.ch/
	// Therefore by default we substract 1 but in a certain case
	// we substract 2
	sub := 1
	// when minOcc is too small we'd risk stripping the wrong nth-child pseudo classes
	if minOcc < 6 {
		sub = 2
	}
	for i := len(lp.path) - sub; i >= 0; i-- {
		if i < iStrip {
			lp.path[i].pseudoClasses = []string{}
		} else if len(lp.path[i].pseudoClasses) > 0 {
			// nth-child(x)
			ncIndex, _ := strconv.Atoi(strings.Replace(strings.Split(lp.path[i].pseudoClasses[0], "(")[1], ")", "", 1))
			if ncIndex >= minOcc {
				lp.path[i].pseudoClasses = []string{}
				iStrip = i
				// we need to pass iStrip to the locationProps too to be used by checkAndUpdateLocProps
				lp.iStrip = iStrip
			}
		}
	}
}

func checkAndUpdateLocProps(old, new *fieldProps) bool {
	// returns true if the paths overlap and the rest of the
	// element location is identical. If true is returned
	// the Selector of old will be updated if necessary.
	if old.textIndex == new.textIndex && old.attr == new.attr {
		if len(old.path) != len(new.path) {
			return false
		}
		newPath := path{}
		for i, on := range old.path {
			if on.tagName == new.path[i].tagName {
				pseudoClassesTmp := []string{}
				if i > old.iStrip {
					pseudoClassesTmp = new.path[i].pseudoClasses
				}
				// the following checks are not complete yet but suffice for now
				// with nth-child being our only pseudo class
				if len(on.pseudoClasses) == len(pseudoClassesTmp) {
					if len(on.pseudoClasses) == 1 {
						if on.pseudoClasses[0] != pseudoClassesTmp[0] {
							return false
						}
					}
					newNode := node{
						tagName:       on.tagName,
						pseudoClasses: on.pseudoClasses,
					}
					if len(on.classes) == 0 && len(new.path[i].classes) == 0 {
						newPath = append(newPath, newNode)
						continue
					}
					ovClasses := utils.IntersectionSlices(on.classes, new.path[i].classes)
					if len(ovClasses) > 0 {
						if i > old.iStrip {
							// if we're past iStrip we only consider nodes equal if they have the same classes
							if len(ovClasses) == len(on.classes) {
								newNode.classes = on.classes
								newPath = append(newPath, newNode)
								continue
							}
						} else {
							// if nodes have more than 0 classes and we're not past iStrip there has to be at least 1 overlapping class
							newNode.classes = ovClasses
							newPath = append(newPath, newNode)
							continue
						}
					}
					// }
				}
			}
			return false

		}
		// if we get until here there is an overlapping path
		old.path = newPath
		old.count++
		old.examples = append(old.examples, new.examples...)
		return true

	}
	return false
}

func filter(l fieldManager, minCount int, removeStaticFields bool) fieldManager {
	// remove if count is smaller than minCount
	// or if the examples are all the same (if removeStaticFields is true)
	i := 0
	for _, p := range l {
		if p.count >= minCount {
			// first reverse the examples list and only take the first x
			utils.ReverseSlice(p.examples)
			p.examples = p.examples[:minCount]
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

func GetDynamicFieldsConfig(s *scraper.Scraper, minOcc int, removeStaticFields bool, modelName, wordsDir string) error {
	if s.URL == "" {
		return errors.New("URL field cannot be empty")
	}
	s.Name = s.URL

	fetcher, err := fetch.NewFetcher(&s.FetcherConfig)
	if err != nil {
		return fmt.Errorf("error creating fetcher: %v", err)
	}

	res, err := fetcher.Fetch(s.URL, fetch.FetchOpts{})
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

	locMan := newFieldManagerFromHtml(htmlStr)
	locMan = squashFieldManager(locMan, minOcc)
	locMan = filter(locMan, minOcc, removeStaticFields)
	locMan.setColors()
	if err := locMan.findFieldNames(modelName, wordsDir); err != nil {
		return err
	}

	if len(locMan) > 0 {
		locMan.selectFieldsTable()
		return locMan.elementsToConfig(s)
	}
	return fmt.Errorf("no fields found")
}
