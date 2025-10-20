package autoconfig

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jakopako/goskyr/utils"
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
// of the fieldProps is identical. If true is returned the path of fp will be updated
// if necessary.
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
							// if we're past iStrip we only consider nodes equal if they have the same classes
							if len(intersectionCls) == len(fpNode.classes) {
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

// squash merges different fieldProps into one
// based on their similarity. The tricky question is 'when are two
// fieldProps close enough to be merged into one?'
func (fm *fieldManager) squash(minOcc int) {
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
			fp.stripNthChild(minOcc)
			squashed = append(squashed, fp)
		}
	}

	*fm = squashed
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
	// level (sibling node) with same tag and the same classes
	for i := range siblingNodes {
		childNode := siblingNodes[i]
		if childNode.tagName == tagName {
			if utils.SliceEquals(childNode.classes, cls) {
				pCls = []string{fmt.Sprintf("nth-child(%d)", len(siblingNodes)+1)}
				break
			}
		}

	}
	return attrs, cls, pCls
}
