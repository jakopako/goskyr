package date

import (
	"errors"
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

func GetDateFormat(dates []string, parts *CoveredDateParts) (string, string) {
	defaultFormat, defaultLanguage := "unknown format. please specify manually", ""
	if len(dates) == 0 {
		return defaultFormat, defaultLanguage
	}
	// only day
	if parts.Day && !parts.Month && !parts.Year && !parts.Time {
		return "2", ""
	}
	// only month
	if parts.Month && !parts.Day && !parts.Year && !parts.Time {
		long := []bool{} // If majority is true then 'January' else 'Jan'
		lang := []string{}
		for _, d := range dates {
			lo, la := findFormatAndLangMonth(d)
			long = append(long, lo)
			lang = append(lang, la)
		}
		isLongFormat := utils.MostOcc(long)
		var monthFormat string
		if isLongFormat {
			monthFormat = "January"
		} else {
			monthFormat = "Jan"
		}
		return monthFormat, utils.MostOcc(lang)
	}
	// only year
	// if parts.Year && !parts.Day && !parts.Month && !parts.Time {

	// }
	// only time
	if parts.Time && !parts.Day && !parts.Month && !parts.Year {
		if strings.Count(dates[0], ":") == 1 {
			return "15:04", ""
		}
	}

	// day, month and time
	if parts.Day && parts.Month && parts.Time && !parts.Year {

	}

	return defaultFormat, defaultLanguage
}

func findFormatAndLangMonth(date string) (bool, string) {
	for l, m := range longMonthNames {
		for n := range m {
			if date == n {
				return true, l
			}
		}
	}
	for l, m := range shortMonthNames {
		for n := range m {
			if date == n {
				return false, l
			}
		}
	}
	return true, ""
}
