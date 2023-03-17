package date

type langMap struct {
	lang     string
	namesMap map[string]bool
}

var longMonthNames = []langMap{
	{
		lang:     "en_US",
		namesMap: longMonthNamesEnUS,
	},
	{
		lang:     "de_DE",
		namesMap: longMonthNamesDeDE,
	},
	{
		lang:     "fr_FR",
		namesMap: longMonthNamesFrFR,
	},
}

var shortMonthNames = []langMap{
	{
		lang:     "en_US",
		namesMap: shortMonthNamesEnUS,
	},
	{
		lang:     "de_DE",
		namesMap: shortMonthNamesDeDE,
	},
	{
		lang:     "fr_FR",
		namesMap: shortMonthNamesFrFR,
	},
}

var shortMonthNamesEnUS = map[string]bool{
	"Jan": true,
	"Feb": true,
	"Mar": true,
	"Apr": true,
	"May": true,
	"Jun": true,
	"Jul": true,
	"Aug": true,
	"Sep": true,
	"Oct": true,
	"Nov": true,
	"Dec": true,
}

var longMonthNamesEnUS = map[string]bool{
	"January":   true,
	"February":  true,
	"March":     true,
	"April":     true,
	"May":       true,
	"June":      true,
	"July":      true,
	"August":    true,
	"September": true,
	"October":   true,
	"November":  true,
	"December":  true,
}

var shortMonthNamesDeDE = map[string]bool{
	"Jan":  true,
	"Feb":  true,
	"Mär":  true,
	"Apr":  true,
	"Mai":  true,
	"Juni": true,
	"Juli": true,
	"Aug":  true,
	"Sep":  true,
	"Okt":  true,
	"Nov":  true,
	"Dez":  true,
}

var longMonthNamesDeDE = map[string]bool{
	"Januar":    true,
	"Februar":   true,
	"März":      true,
	"April":     true,
	"Mai":       true,
	"Juni":      true,
	"Juli":      true,
	"August":    true,
	"September": true,
	"Oktober":   true,
	"November":  true,
	"Dezember":  true,
}

var shortMonthNamesFrFR = map[string]bool{
	"janv": true,
	"févr": true,
	"mars": true,
	"avr":  true,
	"mai":  true,
	"juin": true,
	"juil": true,
	"août": true,
	"sept": true,
	"oct":  true,
	"nov":  true,
	"déc":  true,
}

var longMonthNamesFrFR = map[string]bool{
	"janvier":   true,
	"février":   true,
	"mars":      true,
	"avril":     true,
	"mai":       true,
	"juin":      true,
	"juillet":   true,
	"août":      true,
	"septembre": true,
	"octobre":   true,
	"novembre":  true,
	"décembre":  true,
}
