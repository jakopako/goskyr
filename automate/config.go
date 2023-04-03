package automate

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"github.com/agnivade/levenshtein"
	"github.com/gdamore/tcell/v2"
	"github.com/jakopako/goskyr/date"
	"github.com/jakopako/goskyr/fetch"
	"github.com/jakopako/goskyr/ml"
	"github.com/jakopako/goskyr/scraper"
	"github.com/jakopako/goskyr/utils"
	"github.com/rivo/tview"
	"golang.org/x/net/html"
)

type node struct {
	tagName       string
	classes       []string
	pseudoClasses []string
}

func (n node) string() string {
	nodeString := n.tagName
	for _, cl := range n.classes {
		// https://www.itsupportguides.com/knowledge-base/website-tips/css-colon-in-id/
		cl = strings.ReplaceAll(cl, ":", "\\:")
		cl = strings.ReplaceAll(cl, ">", "\\>")
		// https://stackoverflow.com/questions/45293534/css-class-starting-with-number-is-not-getting-applied
		if unicode.IsDigit(rune(cl[0])) {
			cl = fmt.Sprintf(`\3%s `, string(cl[1:]))
		}
		nodeString += fmt.Sprintf(".%s", cl)
	}
	if len(n.pseudoClasses) > 0 {
		nodeString += fmt.Sprintf(":%s", strings.Join(n.pseudoClasses, ":"))
	}
	return nodeString
}

func (n node) equals(n2 node) bool {
	if n.tagName == n2.tagName {
		if utils.SliceEquals(n.classes, n2.classes) {
			if utils.SliceEquals(n.pseudoClasses, n2.pseudoClasses) {
				return true
			}
		}
	}
	return false
}

type path []node

func (p path) string() string {
	nodeStrings := []string{}
	for _, n := range p {
		nodeStrings = append(nodeStrings, n.string())
	}
	return strings.Join(nodeStrings, " > ")
}

func (p path) distanceTo(p2 path) float64 {
	return float64(levenshtein.ComputeDistance(p.string(), p2.string()))
}

type locationProps struct {
	path       path
	attr       string
	textIndex  int // this will translate into child index within scraper.ElementLocation
	count      int
	examples   []string
	selected   bool
	color      tcell.Color
	distance   float64
	name       string
	stripIndex int // this is needed for the squashLocationManager function
}

type locationManager []*locationProps

func (l locationManager) setColors() {
	if len(l) == 0 {
		return
	}
	for i, e := range l {
		if i != 0 {
			e.distance = l[i-1].distance + l[i-1].path.distanceTo(e.path)
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

func (l locationManager) findFieldNames(modelName, wordsDir string) error {
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

func (l locationManager) selectFieldsTable() {
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

func (l locationManager) elementsToConfig(s *scraper.Scraper) error {
	var locPropsSel []*locationProps
	for _, lm := range l {
		if lm.selected {
			locPropsSel = append(locPropsSel, lm)
		}
	}
	if len(locPropsSel) == 0 {
		return fmt.Errorf("no fields selected")
	}
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
	s.Item = rootSelector.string()
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
			if loc.Attr == "href" {
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

func squashLocationManager(l locationManager, minOcc int) locationManager {
	// This function merges different locationProps into one
	// based on their similarity. The tricky question is 'when are two
	// locationProps close enough to be merged into one?'
	squashed := locationManager{}
	for i := len(l) - 1; i >= 0; i-- {
		lp := l[i]
		updated := false
		for _, sp := range squashed {
			// we need a 'stripIndex' to know which nth-childs we can remove
			// when trying to merge locationProps
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

func stripNthChild(lp *locationProps, minOcc int) {
	borderI := 0
	// a bit arbitrary (and probably not always correct) but
	// for now we assume that borderI cannot be len(lp.path)-1
	// not correct for https://huxleysneuewelt.com/shows
	// but needed for http://www.bar-laparenthese.ch/
	// very hacky:
	sub := 1
	// when minOcc is too small we'd risk stripping the wrong nth-child pseudo classes
	if minOcc < 6 {
		sub = 2
	}
	for i := len(lp.path) - sub; i >= 0; i-- {
		if i < borderI {
			lp.path[i].pseudoClasses = []string{}
		} else if len(lp.path[i].pseudoClasses) > 0 {
			// nth-child(x)
			nc, _ := strconv.Atoi(strings.Replace(strings.Split(lp.path[i].pseudoClasses[0], "(")[1], ")", "", 1))
			if nc >= minOcc {
				lp.path[i].pseudoClasses = []string{}
				borderI = i
				lp.stripIndex = i
			}
		}
	}
}

func checkAndUpdateLocProps(old, new *locationProps) bool {
	// returns true if the paths overlap and the rest of the
	// element location is identical. If true is returned
	// the Selector of a will be updated if necessary.
	if old.textIndex == new.textIndex && old.attr == new.attr {
		if len(old.path) != len(new.path) {
			return false
		}
		newPath := path{}
		for i, on := range old.path {
			if on.tagName == new.path[i].tagName {
				pseudoClassesTmp := []string{}
				if i > old.stripIndex {
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
					// if nodes have more than 0 classes there has to be at least 1 overlapping class
					// does this make sense?
					if len(ovClasses) > 0 {
						newNode.classes = ovClasses
						newPath = append(newPath, newNode)
						continue
					}
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

func filter(l locationManager, minCount int, removeStaticFields bool) locationManager {
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
	allChildren := map[string]int{}    // the nr of children including non-html-tag nodes (ie text)
	tagChildren := map[string][]node{} // the children at the specified nodePath; used for :nth-child() logic
	nodePath := path{}
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
				p := nodePath.string()
				textTrimmed := strings.TrimSpace(text)
				if len(textTrimmed) > 0 {
					ti := allChildren[p]
					lp := locationProps{
						path:      make([]node, len(nodePath)),
						examples:  []string{textTrimmed},
						textIndex: ti,
						count:     1,
					}
					copy(lp.path, nodePath)
					locMan = append(locMan, &lp)
				}
				allChildren[p] += 1
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
					allChildren[nodePath.string()] += 1
					tagChildren[nodePath.string()] = append(tagChildren[nodePath.string()], node{tagName: tnString})
					continue
				}
				if tt == html.StartTagToken {
					allChildren[nodePath.string()] += 1
					tagChildren[nodePath.string()] = append(tagChildren[nodePath.string()], node{tagName: tnString})
					moreAttr := true
					var hrefVal string
					var cls []string
					for moreAttr {
						k, v, m := z.TagAttr()
						vString := strings.TrimSpace(string(v))
						if string(k) == "class" && vString != "" {
							cls = strings.Split(vString, " ")
							j := 0
							for _, cl := range cls {
								// for now we ignore classes that contain dots
								if cl != "" && !strings.Contains(cl, ".") {
									cls[j] = cl
									j++
								}
							}
							cls = cls[:j]
						}
						if string(k) == "href" {
							hrefVal = string(v)
						}
						moreAttr = m
					}
					var pCls []string
					// only add nth-child if there has been another node before at the same
					// level with same tag
					for i := 0; i < len(tagChildren[nodePath.string()])-1; i++ { // the last element is skipped because it's the current node itself
						cn := tagChildren[nodePath.string()][i]
						if cn.tagName == tnString {
							pCls = []string{fmt.Sprintf("nth-child(%d)", len(tagChildren[nodePath.string()]))}
						}

					}
					newNode := node{
						tagName:       tnString,
						classes:       cls,
						pseudoClasses: pCls,
					}
					nodePath = append(nodePath, newNode)
					depth++
					tagChildren[nodePath.string()] = []node{}
					if tnString == "a" && hrefVal != "" {
						lp := locationProps{
							path:     make([]node, len(nodePath)),
							examples: []string{hrefVal},
							attr:     "href",
							count:    1,
						}
						copy(lp.path, nodePath)
						locMan = append(locMan, &lp)
					}
				} else {
					n := true
					for n && depth > 0 {
						if nodePath[len(nodePath)-1].tagName == tnString {
							if tnString == "body" {
								break parse
							}
							n = false
						}
						delete(allChildren, nodePath.string())
						delete(tagChildren, nodePath.string())
						nodePath = nodePath[:len(nodePath)-1]
						depth--
					}
				}
			}
		case html.SelfClosingTagToken:
			if inBody {
				tn, _ := z.TagName()
				tnString := string(tn)
				if tnString == "br" || tnString == "input" || tnString == "img" || tnString == "link" {
					allChildren[nodePath.string()] += 1
					tagChildren[nodePath.string()] = append(tagChildren[nodePath.string()], node{tagName: tnString})
					continue
				}
			}
		}
	}

	locMan = squashLocationManager(locMan, minOcc)
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
