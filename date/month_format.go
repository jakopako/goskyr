package date

var longMonthNames = map[string]map[string]bool{
	"en_US": longMonthNamesEnUS,
	"de_DE": longMonthNamesDeDE,
}

var shortMonthNames = map[string]map[string]bool{
	"en_US": shortMonthNamesEnUS,
	"de_DE": shortMonthNamesDeDE,
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
