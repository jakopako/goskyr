package autoconfig

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/agnivade/levenshtein"
	"github.com/jakopako/goskyr/utils"
)

// A node is our representation of a node in an html tree
type node struct {
	tagName       string
	classes       []string
	pseudoClasses []string
}

// string returns a string representation of the node
func (n node) string() string {
	nodeString := n.tagName
	for _, cl := range n.classes {
		// escape special characters
		// https://www.itsupportguides.com/knowledge-base/website-tips/css-colon-in-id/
		cl = strings.ReplaceAll(cl, ":", "\\:")
		cl = strings.ReplaceAll(cl, ">", "\\>")
		cl = strings.ReplaceAll(cl, "[", "\\[")
		cl = strings.ReplaceAll(cl, "]", "\\]")
		cl = strings.ReplaceAll(cl, "/", "\\/")
		cl = strings.ReplaceAll(cl, "!", "\\!")
		cl = strings.ReplaceAll(cl, "%", "\\%")
		// https://stackoverflow.com/questions/45293534/css-class-starting-with-number-is-not-getting-applied
		if unicode.IsDigit(rune(cl[0])) {
			cl = fmt.Sprintf(`\3%c %s`, cl[0], string(cl[1:]))
		}
		nodeString += fmt.Sprintf(".%s", cl)
	}
	if len(n.pseudoClasses) > 0 {
		nodeString += fmt.Sprintf(":%s", strings.Join(n.pseudoClasses, ":"))
	}
	return nodeString
}

// equals checks if two nodes are equal
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

// A path is a list of nodes starting from the root node and going down
// the html tree to a specific node
type path []node

// string returns a string representation of the path
func (p path) string() string {
	nodeStrings := []string{}
	for _, n := range p {
		nodeStrings = append(nodeStrings, n.string())
	}
	return strings.Join(nodeStrings, " > ")
}

// distance calculates the levenshtein distance between the string represention
// of two paths
func (p path) distance(p2 path) float64 {
	return float64(levenshtein.ComputeDistance(p.string(), p2.string()))
}
