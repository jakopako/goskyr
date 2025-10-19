package autoconfig

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jakopako/goskyr/utils"
	"golang.org/x/net/html"
)

type fieldProps struct {
	path      path
	attr      string
	textIndex int // this will translate into child index within scraper.ElementLocation
	count     int
	examples  []string
	selected  bool
	color     tcell.Color
	distance  float64
	name      string
	iStrip    int // this is needed for the squashLocationManager function
}

func (fp fieldProps) string() string {
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
func newFieldManagerFromHtml(htmlStr string) fieldManager {
	z := html.NewTokenizer(strings.NewReader(htmlStr))
	nodeMgr := fieldManager{}
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
					nodeMgr = append(nodeMgr, &lp)
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
						nodeMgr = append(nodeMgr, &lp)
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
							nodeMgr = append(nodeMgr, &lp)
						}
					}
					continue
				}
			}
		}
	}
	return nodeMgr
}

func (fm fieldManager) equals(fm2 fieldManager) bool {
	if len(fm) != len(fm2) {
		return false
	}

	// sort both fieldManagers for comparison
	slices.SortFunc(fm, func(a, b *fieldProps) int {
		return compareFieldProps(a, b)
	})
	slices.SortFunc(fm2, func(a, b *fieldProps) int {
		return compareFieldProps(a, b)
	})

	for i := range fm {
		if compareFieldProps(fm[i], fm2[i]) != 0 {
			return false
		}
	}
	return true
}

// getTagMetadata, for a given node returns a map of key value pairs (only for the attriutes we're interested in) and
// a list of this node's classes and a list of this node's pseudo classes (currently only nth-child).
func getTagMetadata(tagName string, z *html.Tokenizer, siblingNodes []node) (map[string]string, []string, []string) {
	allowedAttrs := map[string]map[string]bool{
		"a":   {"href": true},
		"img": {"src": true},
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
	for i := 0; i < len(siblingNodes); i++ {
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
