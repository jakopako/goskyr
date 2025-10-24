package autoconfig

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jakopako/goskyr/date"
	"github.com/jakopako/goskyr/ml"
	"github.com/jakopako/goskyr/scraper"
	"github.com/jakopako/goskyr/utils"
	"github.com/rivo/tview"
	"golang.org/x/net/html"
)

type fieldProps struct {
	path      path
	attr      string
	textIndex int // this will translate into child index within scraper.ElementLocation
	count     int // number of occurrences of this fieldProps (might be redundant, because len(examples) could be used)
	examples  []string
	selected  bool
	color     tcell.Color
	distance  float64
	name      string
	iStrip    int // this is needed for the squashLocationManager function
}

func (fp *fieldProps) string() string {
	return fmt.Sprintf("path: %s,\nattr: %s,\ntextIndex: %d,\ncount: %d,\nexamples: %v,\ncolor: %v,\ndistance: %f,\nname: %s,\niStrip: %d\n",
		fp.path.string(),
		fp.attr,
		fp.textIndex,
		fp.count,
		fp.examples,
		fp.color,
		fp.distance,
		fp.name,
		fp.iStrip,
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
				pseudoClassesTmp := []string{}
				if i > fp.iStrip {
					pseudoClassesTmp = other.path[i].pseudoClasses
				}
				// the following checks are not complete yet but suffice for now
				// with nth-child being our only pseudo class
				if len(fpNode.pseudoClasses) == len(pseudoClassesTmp) {
					if len(fpNode.pseudoClasses) == 1 {
						if fpNode.pseudoClasses[0] != pseudoClassesTmp[0] {
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
						if i > fp.iStrip {
							// if we're past iStrip we only consider nodes equal if they have the exact same classes
							// if len(other.path[i].classes) == len(fpNode.classes) && len(intersectionCls) == len(fpNode.classes) {
							if utils.SliceEquals(other.path[i].classes, fpNode.classes) {
								newNode.classes = fpNode.classes
								newPath = append(newPath, newNode)
								continue
							}
						} else {
							// if nodes have more than 0 classes and we're not past iStrip there has to be at least 1 overlapping class
							newNode.classes = intersectionCls
							newPath = append(newPath, newNode)
							continue
						}
					}
				}
			}
			return false
		}
		// if we get until here there is an overlapping path
		fp.path = newPath
		fp.count += other.count
		fp.examples = append(fp.examples, other.examples...)
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
	slices.Sort(fm1.examples)
	slices.Sort(fm2.examples)
	return cmp.Or(
		cmp.Compare(fm1.path.string(), fm2.path.string()),
		cmp.Compare(fm1.attr, fm2.attr),
		cmp.Compare(fm1.textIndex, fm2.textIndex),
		cmp.Compare(fm1.count, fm2.count),
		cmp.Compare(strings.Join(fm1.examples, ","), strings.Join(fm2.examples, ",")),
		cmp.Compare(fm1.name, fm2.name),
		cmp.Compare(fm1.iStrip, fm2.iStrip),
	)
}

// newFieldManagerFromHtml creates a fieldManager by parsing the provided html string
func newFieldManagerFromHtml(htmlStr string) *fieldManager {
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
						examples:  []string{textTrimmed},
						textIndex: ti,
						count:     1,
					}
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
							examples: []string{attrValue},
							attr:     attrKey,
							count:    1,
						}
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
								examples: []string{attrValue},
								attr:     attrKey,
								count:    1,
							}
							copy(lp.path, tmpNodePath)
							fieldMgr = append(fieldMgr, &lp)
						}
					}
					continue
				}
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
// filtering based on minCount and removeStaticFields and setting colors
// and field names
func (fm *fieldManager) process(minCount int, removeStaticFields bool, modelName, wordsDir string) error {
	fm.squash(minCount)
	fm.filter(minCount, removeStaticFields)
	fm.setColors()
	return fm.findFieldNames(modelName, wordsDir)
}

// interactiveFieldSelection shows an interactive table for selecting fields
// and updates the scraper config accordingly
func (fm *fieldManager) interactiveFieldSelection(s *scraper.Scraper) error {
	if len(*fm) == 0 {
		return fmt.Errorf("no fields found")
	}

	if err := fm.showInteractiveTable(); err != nil {
		return err
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
					ss = utils.ShortenString(fm[r-1].examples[c-1], 40)
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

// squash merges different fieldProps into one
// based on their similarity. The tricky question is 'when are two
// fieldProps close enough to be merged into one?'
func (fm *fieldManager) squash(minCount int) {
	squashed := fieldManager{}
	// iterate from the back to ensure that stripNthChild works correctly
	// stripNthChild relies on x in nth-child(x) being >= minOcc which is
	// more likely when iterating from the back over the fieldManager
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
			fp.stripNthChild(minCount)
			squashed = append(squashed, fp)
		}
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
			// first reverse the examples list and only take the first x
			// we reverse because the reverse iteration over the fieldManager
			// in squash resulted in the last examples being first
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

// findFieldNames uses a labler model to predict field names
func (fm fieldManager) findFieldNames(modelName, wordsDir string) error {
	if modelName != "" {
		ll, err := ml.LoadLabler(modelName, wordsDir)
		if err != nil {
			return err
		}
		for _, e := range fm {
			pred, err := ll.PredictLabel(e.examples...)
			if err != nil {
				return err
			}
			e.name = pred // TODO: if label has occured already, add index (eg text-1, text-2...)
		}
	} else {
		for i, e := range fm {
			e.name = fmt.Sprintf("field-%d", i)
		}
	}
	return nil
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
	// level (sibling node) with same tag --and the same classes--
	for i := range siblingNodes {
		childNode := siblingNodes[i]
		if childNode.tagName == tagName {
			// if utils.SliceEquals(childNode.classes, cls) {
			pCls = []string{fmt.Sprintf("nth-child(%d)", len(siblingNodes)+1)}
			break
			// }
		}

	}
	return attrs, cls, pCls
}
