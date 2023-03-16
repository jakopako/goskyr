package date

var longDayNames = map[string]map[string]bool{
	"en_US": longDayNamesEnUS,
	"de_DE": longDayNamesDeDE,
}

var shortDayNames = map[string]map[string]bool{
	"en_US": shortDayNamesEnUS,
	"de_DE": shortDayNamesDeDE,
}

var longDayNamesEnUS = map[string]bool{
	"Sunday":    true,
	"Monday":    true,
	"Tuesday":   true,
	"Wednesday": true,
	"Thursday":  true,
	"Friday":    true,
	"Saturday":  true,
}

var longDayNamesDeDE = map[string]bool{
	"Sonntag":    true,
	"Montag":     true,
	"Dienstag":   true,
	"Mittwoch":   true,
	"Donnerstag": true,
	"Freitag":    true,
	"Samstag":    true,
}

var shortDayNamesDeDE = map[string]bool{
	"So": true,
	"Mo": true,
	"Di": true,
	"Mi": true,
	"Do": true,
	"Fr": true,
	"Sa": true,
}

var shortDayNamesEnUS = map[string]bool{
	"Sun": true,
	"Mon": true,
	"Tue": true,
	"Wed": true,
	"Thu": true,
	"Fri": true,
	"Sat": true,
}
