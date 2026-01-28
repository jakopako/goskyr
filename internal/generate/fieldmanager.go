package generate

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jakopako/goskyr/internal/date"
	"github.com/jakopako/goskyr/internal/scraper"
	"github.com/jakopako/goskyr/internal/utils"
	"github.com/rivo/tview"
	"golang.org/x/net/html"
)

type fieldProps struct {
	path      path
	attr      string
	textIndex int // this will translate into child index within scraper.ElementLocation
	count     int // number of occurrences of this fieldProps (might be redundant, because len(examples) could be used)
	examples  []fieldExample
	selected  bool
	color     tcell.Color
	distance  float64
	name      string
	iStrip    int // this is needed for the squashLocationManager function
	origI     int // original index in fieldManager before sorting by iStrip
}

type fieldExample struct {
	example string
	origI   int
}

func (fp *fieldProps) string() string {
	return fmt.Sprintf("path: %s,\nattr: %s,\ntextIndex: %d,\ncount: %d,\nexamples: %v,\ncolor: %v,\ndistance: %f,\nname: %s,\niStrip: %d,\norigI: %d\n",
		fp.path.string(),
		fp.attr,
		fp.textIndex,
		fp.count,
		fp.examples,
		fp.color,
		fp.distance,
		fp.name,
		fp.iStrip,
		fp.origI,
	)
}

// checkOverlapAndUpdate checks if the paths of fp and other overlap and if the rest
// of the fieldProps is identical. If true is returned the path, examples & count
// of fp will be updated if necessary.
func (fp *fieldProps) checkOverlapAndUpdate(other *fieldProps) bool {
	if fp.textIndex == other.textIndex && fp.attr == other.attr {
		if len(fp.path) != len(other.path) {
			return false
		}
		newPath := path{}
		for i, fpNode := range fp.path {
			if fpNode.tagName == other.path[i].tagName {
				if i > fp.iStrip {
					// the following checks are not complete yet but suffice for now
					// with nth-child being our only pseudo class
					if len(fpNode.pseudoClasses) == len(other.path[i].pseudoClasses) {
						if len(fpNode.pseudoClasses) == 1 {
							if fpNode.pseudoClasses[0] != other.path[i].pseudoClasses[0] {
								return false
							}
						}
					} else {
						return false
					}
				}

				newNode := node{
					tagName:       fpNode.tagName,
					pseudoClasses: fpNode.pseudoClasses,
				}
				if len(fpNode.classes) == 0 && len(other.path[i].classes) == 0 {
					newPath = append(newPath, newNode)
					continue
				}
				intersectionCls := utils.IntersectionSlices(fpNode.classes, other.path[i].classes)
				if len(intersectionCls) > 0 {
					newNode.classes = intersectionCls
					newPath = append(newPath, newNode)
					continue
				}
			}
			return false
		}
		// if we get until here there is an overlapping path
		fp.path = newPath
		fp.count += other.count
		fp.examples = append(fp.examples, other.examples...)
		fp.origI = min(fp.origI, other.origI)
		return true
	}
	return false
}

// stripNthChild tries to find the index in a fieldProps path under which
// we need to strip the nth-child pseudo class. We need to strip that pseudo
// class because at a later point we want to find a common base path between
// different paths but if all paths' base paths look differently (because their
// nodes have different nth-child pseudo classes) there won't be a common
// base path.
func (fp *fieldProps) stripNthChild(minOcc int) {
	iStrip := 0
	// every node in fp.path with index < than iStrip needs no be stripped
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
	for i := len(fp.path) - sub; i >= 0; i-- {
		if i < iStrip {
			fp.path[i].pseudoClasses = []string{}
		} else if len(fp.path[i].pseudoClasses) > 0 {
			// nth-child(x)
			nthChildI, _ := strconv.Atoi(strings.Replace(strings.Split(fp.path[i].pseudoClasses[0], "(")[1], ")", "", 1))
			if nthChildI >= minOcc {
				fp.path[i].pseudoClasses = []string{}
				iStrip = i
				// we need to pass iStrip to the fieldProps too to be used by checkOverlapAndUpdate
				fp.iStrip = iStrip
			}
		}
	}
}

type fieldManager []*fieldProps

func (fm fieldManager) string() string {
	result := ""
	for i, fp := range fm {
		result += fp.string()
		if i < len(fm)-1 {
			result += "\n"
		}
	}
	return result
}

// compareFieldProps compares two fieldProps and returns an int indicating their order
func compareFieldProps(fm1, fm2 *fieldProps) int {
	// for now we ignore 'selected', 'color' & 'distance' in comparison

	cmps := []int{
		cmp.Compare(fm1.path.string(), fm2.path.string()),
		cmp.Compare(fm1.attr, fm2.attr),
		cmp.Compare(fm1.textIndex, fm2.textIndex),
		cmp.Compare(fm1.count, fm2.count),
		// cmp.Compare(strings.Join(fm1.examples, ","), strings.Join(fm2.examples, ",")),
		cmp.Compare(len(fm1.examples), len(fm2.examples)),
		cmp.Compare(fm1.name, fm2.name),
		cmp.Compare(fm1.iStrip, fm2.iStrip),
		cmp.Compare(fm1.origI, fm2.origI),
	}

	// not sure if this makes sense
	for i := 0; i < min(len(fm1.examples), len(fm2.examples)); i++ {
		cmps = append(cmps, cmp.Compare(fm1.examples[i].example, fm2.examples[i].example))
		cmps = append(cmps, cmp.Compare(fm1.examples[i].origI, fm2.examples[i].origI))
	}

	return cmp.Or(cmps...)
}

// newFieldManagerFromHtml creates a fieldManager by parsing the provided html string
func newFieldManagerFromHtml(htmlStr string) *fieldManager {
	// add index to fieldProps so we can eventually
	// sort them back to original order
	index := 0
	z := html.NewTokenizer(strings.NewReader(htmlStr))
	fieldMgr := fieldManager{}
	nrChildren := map[string]int{}    // the nr of children a node (represented by a path) has, including non-html-tag nodes (ie text)
	childNodes := map[string][]node{} // the children of the node at the specified nodePath; used for :nth-child() logic
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
					ti := nrChildren[p]
					lp := fieldProps{
						path:      make([]node, len(nodePath)),
						examples:  []fieldExample{{example: textTrimmed, origI: index}},
						textIndex: ti,
						count:     1,
						origI:     index,
					}
					index++
					copy(lp.path, nodePath)
					fieldMgr = append(fieldMgr, &lp)
				}
				nrChildren[p] += 1
			}
		case html.StartTagToken, html.EndTagToken:
			tn, _ := z.TagName()
			tagNameStr := string(tn)
			if tagNameStr == "body" {
				inBody = !inBody
			}
			if inBody {
				// br can also be self closing tag, see later case statement
				if tagNameStr == "br" || tagNameStr == "input" {
					nrChildren[nodePath.string()] += 1
					childNodes[nodePath.string()] = append(childNodes[nodePath.string()], node{tagName: tagNameStr})
					continue
				}
				if tt == html.StartTagToken {
					attrs, cls, pCls := getTagMetadata(tagNameStr, z, childNodes[nodePath.string()])
					nrChildren[nodePath.string()] += 1
					childNodes[nodePath.string()] = append(childNodes[nodePath.string()], node{tagName: tagNameStr, classes: cls})

					newNode := node{
						tagName:       tagNameStr,
						classes:       cls,
						pseudoClasses: pCls,
					}
					nodePath = append(nodePath, newNode)
					depth++
					childNodes[nodePath.string()] = []node{}

					for attrKey, attrValue := range attrs {
						lp := fieldProps{
							path:     make([]node, len(nodePath)),
							examples: []fieldExample{{example: attrValue, origI: index}},
							attr:     attrKey,
							count:    1,
							origI:    index,
						}
						index++
						copy(lp.path, nodePath)
						fieldMgr = append(fieldMgr, &lp)
					}
				} else {
					n := true
					for n && depth > 0 {
						if nodePath[len(nodePath)-1].tagName == tagNameStr {
							if tagNameStr == "body" {
								break parse
							}
							n = false
						}
						delete(nrChildren, nodePath.string())
						delete(childNodes, nodePath.string())
						nodePath = nodePath[:len(nodePath)-1]
						depth--
					}
				}
			}
		case html.SelfClosingTagToken:
			if inBody {
				tn, _ := z.TagName()
				tagNameStr := string(tn)
				if tagNameStr == "br" || tagNameStr == "input" || tagNameStr == "img" || tagNameStr == "link" {
					attrs, cls, pCls := getTagMetadata(tagNameStr, z, childNodes[nodePath.string()])
					nrChildren[nodePath.string()] += 1
					childNodes[nodePath.string()] = append(childNodes[nodePath.string()], node{tagName: tagNameStr, classes: cls})

					if len(attrs) > 0 {
						tmpNodePath := make([]node, len(nodePath))
						copy(tmpNodePath, nodePath)
						newNode := node{
							tagName:       tagNameStr,
							classes:       cls,
							pseudoClasses: pCls,
						}
						tmpNodePath = append(tmpNodePath, newNode)

						for attrKey, attrValue := range attrs {
							lp := fieldProps{
								path:     make([]node, len(tmpNodePath)),
								examples: []fieldExample{{example: attrValue, origI: index}},
								attr:     attrKey,
								count:    1,
								origI:    index,
							}
							index++
							copy(lp.path, tmpNodePath)
							fieldMgr = append(fieldMgr, &lp)
						}
					}
					continue
				}
			}
		case html.CommentToken:
			// ignore comments but increase child count
			if inBody {
				nrChildren[nodePath.string()] += 1
			}
		}
	}
	return &fieldMgr
}

// equals checks if two fieldManagers are equal
func (fm *fieldManager) equals(fm2 *fieldManager) bool {
	if len(*fm) != len(*fm2) {
		return false
	}

	// sort both fieldManagers for comparison
	slices.SortFunc(*fm, func(a, b *fieldProps) int {
		return compareFieldProps(a, b)
	})
	slices.SortFunc(*fm2, func(a, b *fieldProps) int {
		return compareFieldProps(a, b)
	})

	for i := range *fm {
		if compareFieldProps((*fm)[i], (*fm2)[i]) != 0 {
			return false
		}
	}
	return true
}

// process processes the fieldManager by squashing similar fieldProps,
// filtering based the config and setting colors and field names
func (fm *fieldManager) process(config *Config) error {
	fm.squash(config.MinOccurrences)
	fm.filter(config.MinOccurrences, config.DistinctValues)
	fm.setColors()
	return fm.labelFields(&config.LablerConfig)
}

// fieldSelection either shows an interactive table for selecting fields (interactive=true)
// or simply selects all fields (interactive=false) and consequently updates the scraper config accordingly
func (fm *fieldManager) fieldSelection(s *scraper.Scraper, interactive bool) error {
	if len(*fm) == 0 {
		return fmt.Errorf("no fields found")
	}

	if !interactive {
		for _, fp := range *fm {
			fp.selected = true
		}
	} else {
		if err := fm.showInteractiveTable(); err != nil {
			return err
		}
	}

	return fm.elementsToConfig(s)
}

// showInteractiveTable shows an interactive table for selecting fields
func (fm fieldManager) showInteractiveTable() error {
	app := tview.NewApplication()
	table := tview.NewTable().SetBorders(true)
	cols, rows := 5, len(fm)+1
	for r := range rows {
		for c := range cols {
			color := tcell.ColorWhite
			if c < 1 || r < 1 {
				if c < 1 && r > 0 {
					color = tcell.ColorGreen
					table.SetCell(r, c, tview.NewTableCell(fmt.Sprintf("[%d] %s", r-1, fm[r-1].name)).
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
				if len(fm[r-1].examples) >= c {
					ss = utils.ShortenString(fm[r-1].examples[c-1].example, 40)
				}
				table.SetCell(r, c,
					tview.NewTableCell(ss).
						SetTextColor(fm[r-1].color).
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
		fm[row-1].selected = !fm[row-1].selected
		if fm[row-1].selected {
			table.GetCell(row, 0).SetTextColor(tcell.ColorRed)
			for i := 1; i < 5; i++ {
				table.GetCell(row, i).SetTextColor(tcell.ColorOrange)
			}
		} else {
			table.GetCell(row, 0).SetTextColor(tcell.ColorGreen)
			for i := 1; i < 5; i++ {
				table.GetCell(row, i).SetTextColor(fm[row-1].color)
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
		return err
	}
	return nil
}

// elementsToConfig converts the selected fieldProps into scraper config
func (fm fieldManager) elementsToConfig(s *scraper.Scraper) error {
	// remove unselected fieldProps
	j := 0
	for i, fp := range fm {
		if fp.selected {
			fm[j] = fm[i]
			j++
		}
	}

	fm = fm[:j]
	if len(fm) == 0 {
		return fmt.Errorf("no fields selected")
	}

	// find shared root selector
	var rootSelector path
outer:
	for i := 0; ; i++ {
		var n node
		for j, e := range fm {
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
	s.Item = rootSelector.trimPrefix(3).string()

	// for now we assume that there will only be one date field
	t := time.Now()
	zone, _ := t.Zone()
	zone = strings.Replace(zone, "CEST", "CET", 1) // quick hack for issue #209
	dateField := scraper.Field{
		Name:         "date",
		Type:         "date",
		DateLocation: zone,
	}
	for _, e := range fm {
		exampleStrs := []string{}
		for _, ex := range e.examples {
			exampleStrs = append(exampleStrs, ex.example)
		}
		loc := scraper.ElementLocation{
			Selector:   e.path[len(rootSelector):].string(),
			ChildIndex: e.textIndex,
			Attr:       e.attr,
			Examples:   exampleStrs[:4],
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
			format, lang := date.GetDateFormatMulti(exampleStrs, cd)
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

// squash merges different fieldProps into one
// based on their similarity. The tricky question is 'when are two
// fieldProps close enough to be merged into one?'
func (fm *fieldManager) squash(minCount int) {
	squashed := fieldManager{}

	// first compute iStrip for all fieldProps
	for _, fp := range *fm {
		fp.stripNthChild(minCount)
	}

	// now sort by iStrip ascending to ensure that fieldProps with
	// higher iStrip are processed first in the following loop
	// this ensures that fieldProps with low iStrip are more likely
	// to be merged into existing squashed fieldProps
	slices.SortFunc(*fm, func(a, b *fieldProps) int {
		return a.iStrip - b.iStrip
	})

	for i := len(*fm) - 1; i >= 0; i-- {
		fp := (*fm)[i]
		updated := false
		for _, sfp := range squashed {
			updated = sfp.checkOverlapAndUpdate(fp)
			if updated {
				break
			}
		}
		if !updated {
			squashed = append(squashed, fp)
		}
	}

	// finally sort back to original order
	slices.SortFunc(squashed, func(a, b *fieldProps) int {
		return a.origI - b.origI
	})

	// also, sort examples within each fieldProps by their origI
	for _, fp := range squashed {
		slices.SortFunc(fp.examples, func(a, b fieldExample) int {
			return a.origI - b.origI
		})
	}

	*fm = squashed
}

// filter removes fieldProps that do not meet certain criteria
func (fm *fieldManager) filter(minCount int, removeStaticFields bool) {
	// remove if count is smaller than minCount
	// or if the examples are all the same (if removeStaticFields is true)
	i := 0
	for _, p := range *fm {
		if p.count >= minCount {
			p.examples = p.examples[:minCount]
			if removeStaticFields {
				eqEx := true
				for _, ex := range p.examples {
					if ex.example != p.examples[0].example {
						eqEx = false
						break
					}
				}
				if !eqEx {
					(*fm)[i] = p
					i++
				}
			} else {
				(*fm)[i] = p
				i++
			}
		}
	}
	*fm = (*fm)[:i]
}

// setColors computes a color for each fieldProps based on its distance
// from the first fieldProps in the fieldManager
func (fm fieldManager) setColors() {
	if len(fm) == 0 {
		return
	}
	for i, e := range fm {
		if i != 0 {
			e.distance = fm[i-1].distance + fm[i-1].path.distance(e.path)
		}
	}
	// scale to 1 and map to rgb
	maxDist := fm[len(fm)-1].distance * 1.2
	s := 0.73
	v := 0.96
	for _, e := range fm {
		h := e.distance / maxDist
		r, g, b := utils.HSVToRGB(h, s, v)
		e.color = tcell.NewRGBColor(r, g, b)
	}
}

// labelFields uses a labler to predict field names
func (fm fieldManager) labelFields(lc *LablerConfig) error {
	labler, err := newLabler(lc)
	if err != nil {
		return err
	}

	return labler.labelFields(fm)
}

// getTagMetadata, for a given node returns a map of key value pairs (only for the attriutes we're interested in) and
// a list of this node's classes and a list of this node's pseudo classes (currently only nth-child).
func getTagMetadata(tagName string, z *html.Tokenizer, siblingNodes []node) (map[string]string, []string, []string) {
	allowedAttrs := map[string]map[string]bool{
		"a":   {"href": true, "title": true},
		"img": {"src": true, "title": true},
	}
	moreAttr := true
	attrs := make(map[string]string)
	var cls []string       // classes
	if tagName != "body" { // we don't care about classes for the body tag
		for moreAttr {
			k, v, m := z.TagAttr()
			vString := strings.TrimSpace(string(v))
			kString := string(k)
			if kString == "class" && vString != "" {
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
			if _, found := allowedAttrs[tagName]; found {
				if _, found := allowedAttrs[tagName][kString]; found {
					attrs[kString] = vString
				}
			}
			moreAttr = m
		}
	}
	var pCls []string // pseudo classes
	// only add nth-child if there has been another node before at the same
	// level (sibling node) with same tag
	for i := range siblingNodes {
		childNode := siblingNodes[i]
		if childNode.tagName == tagName {
			pCls = []string{fmt.Sprintf("nth-child(%d)", len(siblingNodes)+1)}
			break
		}

	}
	return attrs, cls, pCls
}
