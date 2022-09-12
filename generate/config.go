package automate

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jakopako/goskyr/scraper"
	"github.com/jakopako/goskyr/utils"
	"golang.org/x/net/html"
)

type locationProps struct {
	loc      scraper.ElementLocation
	count    int
	examples []string
}

type locationManager []*locationProps

func update(l locationManager, e scraper.ElementLocation, s string) locationManager {
	// updates count and examples or adds new element to the locationManager
	// old implementation
	// if p, found := (*l)[e]; found {
	// 	p.count += 1
	// 	if p.count <= 4 {
	// 		p.examples = append(p.examples, s)
	// 	}
	// } else {
	// 	(*l)[e] = &locationProps{count: 1, examples: []string{s}}
	// }

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
	if a.NodeIndex == b.NodeIndex && a.ChildIndex == b.ChildIndex {
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
	i := 0
	for _, p := range l {
		if p.count >= minCount {
			l[i] = p
			i++
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
		d := scraper.DynamicField{
			Name:            fmt.Sprintf("field-%d", i),
			Type:            "text",
			ElementLocation: e,
		}
		s.Fields.Dynamic = append(s.Fields.Dynamic, d)
	}
}

func GetDynamicFieldsConfig(s *scraper.Scraper, minOcc int) error {
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
						moreAttr = m
					}
					if tnString != "br" {
						nodePath = append(nodePath, tnString)
						nrChildren[pathToSelector(nodePath)] = 0
						depth++
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

		colorReset := "\033[0m"
		colorGreen := "\033[32m"
		colorBlue := "\033[34m"
		for i, e := range locMan {
			fmt.Printf("%sfield [%d]%s\n  %slocation:%s %+v\n  %sexamples:%s\n\t%s\n\n", colorGreen, i, colorReset, colorBlue, colorReset, e.loc, colorBlue, colorReset, strings.Join(e.examples, "\n\t"))
		}

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
