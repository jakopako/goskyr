package date

var longDayNames = []langMap{
	{
		lang:     "en_US",
		namesMap: longDayNamesEnUS,
	},
	{
		lang:     "de_DE",
		namesMap: longDayNamesDeDE,
	},
	{
		lang:     "fr_FR",
		namesMap: longDayNamesFrFR,
	},
	{
		lang:     "nl_BE",
		namesMap: longDayNamesNlBE,
	},
	{
		lang:     "sk_SK",
		namesMap: longDayNamesSkSK,
	},
}

var shortDayNames = []langMap{
	{
		lang:     "en_US",
		namesMap: shortDayNamesEnUS,
	},
	{
		lang:     "de_DE",
		namesMap: shortDayNamesDeDE,
	},
	{
		lang:     "fr_FR",
		namesMap: shortDayNamesFrFR,
	},
	{
		lang:     "nl_BE",
		namesMap: shortDayNamesNlBE,
	},
	{
		lang:     "sk_SK",
		namesMap: shortDayNamesSkSK,
	},
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

var longDayNamesEnUS = map[string]bool{
	"Sunday":    true,
	"Monday":    true,
	"Tuesday":   true,
	"Wednesday": true,
	"Thursday":  true,
	"Friday":    true,
	"Saturday":  true,
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

var longDayNamesDeDE = map[string]bool{
	"Sonntag":    true,
	"Montag":     true,
	"Dienstag":   true,
	"Mittwoch":   true,
	"Donnerstag": true,
	"Freitag":    true,
	"Samstag":    true,
}

var shortDayNamesFrFR = map[string]bool{
	"dim": true,
	"lun": true,
	"mar": true,
	"mer": true,
	"jeu": true,
	"ven": true,
	"sam": true,
}

var longDayNamesFrFR = map[string]bool{
	"dimanche": true,
	"lundi":    true,
	"mardi":    true,
	"mercredi": true,
	"jeudi":    true,
	"vendredi": true,
	"samedi":   true,
}

var shortDayNamesNlBE = map[string]bool{
	"zo": true,
	"ma": true,
	"di": true,
	"wo": true,
	"do": true,
	"vr": true,
	"za": true,
}

var longDayNamesNlBE = map[string]bool{
	"zondag":    true,
	"maandag":   true,
	"dinsdag":   true,
	"woensdag":  true,
	"donderdag": true,
	"vrijdag":   true,
	"zaterdag":  true,
}

var shortDayNamesSkSK = map[string]bool{
	"ne": true,
	"po": true,
	"ut": true,
	"st": true,
	"št": true,
	"pi": true,
	"so": true,
}

var longDayNamesSkSK = map[string]bool{
	"nedeľa":   true,
	"pondelok": true,
	"utorok":   true,
	"streda":   true,
	"štvrtok":  true,
	"piatok":   true,
	"sobota":   true,
}
