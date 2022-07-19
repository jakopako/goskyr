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
	count    int
	examples []string
}

type locationManager map[scraper.ElementLocation]*locationProps

func (l *locationManager) update(e scraper.ElementLocation, s string) {
	// updates count and examples or adds new element to the locationManager
	// old implementation
	if p, found := (*l)[e]; found {
		p.count += 1
		if p.count <= 4 {
			p.examples = append(p.examples, s)
		}
	} else {
		(*l)[e] = &locationProps{count: 1, examples: []string{s}}
	}
	// TODO: new implementation
}

func (l *locationManager) filter(minCount int) {
	for e, p := range *l {
		if p.count < minCount {
			delete(*l, e)
		}
	}
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
					itemSelector = pathToSelector(selectorToPath(e.Selector)[:i-1])
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
	locOcc := locationManager{}
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
					// this check has to be updated
					// we cannot check the exact path because a repeating node might have both
					// repeating classes and non-repeating classes. So instead of checking whether
					// we have seen the exact location we need to check whether there is a location
					// where for each node in the path that there is at least on overlapping class
					// (if at least one of the two nodes has a class)
					locOcc.update(l, strings.TrimSpace(text))
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

	locOcc.filter(minOcc)

	if len(locOcc) > 0 {
		f := make([]scraper.ElementLocation, len(locOcc))
		i := 0
		for k := range locOcc {
			f[i] = k
			i++
		}
		sort.Slice(f, func(p, q int) bool {
			return f[p].Selector > f[q].Selector
		})

		colorReset := "\033[0m"
		colorGreen := "\033[32m"
		colorBlue := "\033[34m"
		for i, e := range f {
			fmt.Printf("%sfield [%d]%s\n  %slocation:%s %+v\n  %sexamples:%s\n\t%s\n\n", colorGreen, i, colorReset, colorBlue, colorReset, e, colorBlue, colorReset, strings.Join(locOcc[e].examples, "\n\t"))
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
		var fs []scraper.ElementLocation
		for _, n := range ns {
			if n >= len(f) {
				return fmt.Errorf("please enter valid numbers")
			}
			fs = append(fs, f[n])
		}

		elementsToConfig(s, fs...)
		return nil
	}
	return fmt.Errorf("no fields found")
}
