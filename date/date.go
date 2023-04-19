package date

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jakopako/goskyr/utils"
)

// CoveredDateParts is used to determine what parts of a date a
// DateComponent covers
type CoveredDateParts struct {
	Day   bool `yaml:"day,omitempty"`
	Month bool `yaml:"month,omitempty"`
	Year  bool `yaml:"year,omitempty"`
	Time  bool `yaml:"time,omitempty"`
}

func CheckForDoubleDateParts(dpOne CoveredDateParts, dpTwo CoveredDateParts) error {
	if dpOne.Day && dpTwo.Day {
		return errors.New("date parsing error: 'day' covered at least twice")
	}
	if dpOne.Month && dpTwo.Month {
		return errors.New("date parsing error: 'month' covered at least twice")
	}
	if dpOne.Year && dpTwo.Year {
		return errors.New("date parsing error: 'year' covered at least twice")
	}
	if dpOne.Time && dpTwo.Time {
		return errors.New("date parsing error: 'time' covered at least twice")
	}
	return nil
}

func MergeDateParts(dpOne CoveredDateParts, dpTwo CoveredDateParts) CoveredDateParts {
	return CoveredDateParts{
		Day:   dpOne.Day || dpTwo.Day,
		Month: dpOne.Month || dpTwo.Month,
		Year:  dpOne.Year || dpTwo.Year,
		Time:  dpOne.Time || dpTwo.Time,
	}
}

func HasAllDateParts(cdp CoveredDateParts) bool {
	return cdp.Day && cdp.Month && cdp.Year && cdp.Time
}

func GetDateFormatMulti(dates []string, parts CoveredDateParts) (string, string) {
	fs, ls := []string{}, []string{}
	for _, d := range dates {
		f, l := GetDateFormat(d, parts)
		fs = append(fs, f)
		ls = append(ls, l)
	}
	return utils.MostOcc(fs), utils.MostOcc(ls)
}

func GetDateFormat(date string, parts CoveredDateParts) (string, string) {
	defaultFormat, defaultLanguage := "unknown format. please specify manually", ""
	if len(date) == 0 {
		return defaultFormat, defaultLanguage
	}

	separators := []rune{' ', ',', '.', '-', ':'}

	tokens := []string{}
	sepTokens := []string{}

	currToken := ""
	// split date into tokens. Tokens are strings of characters, separators are single separator characters.
	for _, c := range date {
		if utils.RuneIsOneOf(c, separators) {
			if currToken != "" || len(tokens) == 0 {
				// previous c was no separator
				tokens = append(tokens, currToken)
				currToken = ""
				sepTokens = append(sepTokens, string(c))
			} else {
				// previous c was also a separator or we are at the beginning
				tokens = append(tokens, "")
				sepTokens = append(sepTokens, string(c))
			}
		} else {
			currToken += string(c)
		}
	}
	// push last tokens to respective arrays
	if currToken != "" {
		tokens = append(tokens, currToken)
	}
	// make sure both arrays have the same length
	if len(sepTokens) < len(tokens) {
		sepTokens = append(sepTokens, "")
	}

	potLangs := [][]string{}
	formatTokens := []string{}
	for i, token := range tokens {
		if token == "" {
			formatTokens = append(formatTokens, token)
			continue
		}
		if !utils.ContainsDigits(token) {
			if parts.Month {
				if m, l, err := getFormatAndLangMonthLetters(token); err == nil {
					formatTokens = append(formatTokens, m)
					potLangs = append(potLangs, l)
					parts.Month = false // so that we know that we had month already
					continue
				}
			}
			if parts.Day {
				if d, l, err := getFormatAndLangDayLetters(token); err == nil {
					formatTokens = append(formatTokens, d)
					potLangs = append(potLangs, l)
					// in contrast to month we don't do this with day because it happens
					// that day occurs as number _and_ as word in a single date
					// parts.Day = false
					continue
				}
			}
		} else {
			if parts.Day {
				if isDayNumber(token) {
					formatTokens = append(formatTokens, "2")
					parts.Day = false // we might have to remove this line in the future depending on what dates we'll encounter
					continue
				}
			}
			if parts.Month {
				if isMonthNumber(token) {
					formatTokens = append(formatTokens, "1")
					parts.Month = false
					continue
				}
			}
			if parts.Year {
				if yf, err := getYearFormatPart(token); err == nil {
					formatTokens = append(formatTokens, yf)
					parts.Year = false
					continue
				}
			}
			if parts.Time {
				if tf, err := getTimeFormatPart(i, sepTokens, tokens); err == nil {
					formatTokens = append(formatTokens, tf)
					continue
				}
			}
		}
		formatTokens = append(formatTokens, token)
	}

	// putting everything together
	finalFormat := ""
	for i, ft := range formatTokens {
		finalFormat += ft
		finalFormat += sepTokens[i]
	}

	// finding the correct language
	language := ""
	if len(potLangs) > 1 {
		intersection := potLangs[0]
		for i := 1; i < len(potLangs) && len(intersection) > 0; i++ {
			intersection = utils.IntersectionSlices(intersection, potLangs[i])
		}
		if len(intersection) > 0 {
			language = intersection[0]
		}
	} else if len(potLangs) > 0 {
		language = potLangs[0][0]
	}
	return finalFormat, language
}

func getFormatAndLangMonthLetters(month string) (string, []string, error) {
	potLangs := []string{}
	monthTmp := strings.ToLower(month)
	for _, m := range longMonthNames {
		for n := range m.namesMap {
			if monthTmp == strings.ToLower(n) {
				potLangs = append(potLangs, m.lang)
			}
		}
	}
	if len(potLangs) > 0 {
		return "January", potLangs, nil
	}
	for _, m := range shortMonthNames {
		for n := range m.namesMap {
			if monthTmp == strings.ToLower(n) {
				potLangs = append(potLangs, m.lang)
			}
		}
	}
	if len(potLangs) > 0 {
		return "Jan", potLangs, nil
	}
	return "", potLangs, fmt.Errorf("%s is not a month", month)
}

func getFormatAndLangDayLetters(day string) (string, []string, error) {
	potLangs := []string{} // there might be multiple matches for a certain day string
	dayTmp := strings.ToLower(day)
	for _, m := range longDayNames {
		for n := range m.namesMap {
			if dayTmp == strings.ToLower(n) {
				potLangs = append(potLangs, m.lang)
			}
		}
	}
	if len(potLangs) > 0 {
		return "Monday", potLangs, nil
	}
	for _, m := range shortDayNames {
		for n := range m.namesMap {
			if dayTmp == strings.ToLower(n) {
				potLangs = append(potLangs, m.lang)
			}
		}
	}
	if len(potLangs) > 0 {
		return "Mon", potLangs, nil
	}
	return "", potLangs, fmt.Errorf("%s is not a day", day)
}

func isDayNumber(number string) bool {
	// this function will have to be improved once we have more different date formats
	if len(number) <= 2 && utils.OnlyContainsDigits(number) {
		return true
	}
	return false
}

func isMonthNumber(number string) bool {
	if len(number) <= 2 && utils.OnlyContainsDigits(number) {
		return true
	}
	return false
}

func getTimeFormatPart(index int, sepTokens []string, tokens []string) (string, error) {
	if len(tokens[index]) <= 2 {
		if sepTokens[index] == ":" || sepTokens[index] == "." {
			// hour
			return "15", nil
		}
		if index > 0 {
			if sepTokens[index-1] == ":" || sepTokens[index-1] == "." {
				// minute (could also be second but haven't encountered it so far. Adapt when necessary)
				return "04", nil
			}
		}
		if len(tokens) > index+1 {
			if tokens[index+1] == "Uhr" {
				return "15", nil
			}
		}
	} else {
		// one of 04h, 15u04, 15h04, 04pm, 15pm
		if strings.HasSuffix(tokens[index], "h") {
			return "04h", nil
		}
		if strings.HasSuffix(strings.ToLower(tokens[index]), "pm") || strings.HasSuffix(strings.ToLower(tokens[index]), "am") {
			suffix := tokens[index][len(tokens[index])-2:]
			isUpper := suffix == "PM" || suffix == "AM"
			suffixFormatted := "pm"
			if isUpper {
				suffixFormatted = "PM"
			}
			if index > 0 {
				if sepTokens[index-1] != " " {
					return fmt.Sprintf("04%s", suffixFormatted), nil
				}
			}
			return fmt.Sprintf("15%s", suffixFormatted), nil
		}
		if strings.Contains(tokens[index], "u") {
			return "15u04", nil
		}
		if strings.Contains(tokens[index], "h") {
			return "15h04", nil
		}
	}
	return "", fmt.Errorf("%s is not (part of) a time string", tokens[index])
}

func getYearFormatPart(token string) (string, error) {
	if len(token) == 4 {
		return "2006", nil
	}
	if len(token) == 2 {
		return "06", nil
	}
	return "", fmt.Errorf("%s is not a year string", token)
}
