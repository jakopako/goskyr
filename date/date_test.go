package date

import (
	"log"
	"testing"
)

type formatTestStruct struct {
	input        string
	coveredParts CoveredDateParts
	formatString string
	language     string
}

func TestGetDateFormat1(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "March",
			coveredParts: CoveredDateParts{Month: true},
			formatString: "January",
			language:     "en_US",
		},
		{
			input:        "February",
			coveredParts: CoveredDateParts{Month: true},
			formatString: "January",
			language:     "en_US",
		},
		{
			input:        "April",
			coveredParts: CoveredDateParts{Month: true},
			formatString: "January",
			language:     "en_US",
		},
		{
			input:        "May",
			coveredParts: CoveredDateParts{Month: true},
			formatString: "January",
			language:     "en_US",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat2(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "3",
			coveredParts: CoveredDateParts{Day: true},
			formatString: "2",
		},
		{
			input:        "04",
			coveredParts: CoveredDateParts{Day: true},
			formatString: "2",
		},
		{
			input:        "16",
			coveredParts: CoveredDateParts{Day: true},
			formatString: "2",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat3(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "19:45",
			coveredParts: CoveredDateParts{Time: true},
			formatString: "15:04",
		},
		{
			input:        "23:30",
			coveredParts: CoveredDateParts{Time: true},
			formatString: "15:04",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat4(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "Wednesday, 1 march om 21u00",
			coveredParts: CoveredDateParts{Day: true, Month: true, Time: true},
			formatString: "Monday, 2 January om 15u04",
			language:     "en_US",
		},
		{
			input:        "Thursday, 2 march om 21u00",
			coveredParts: CoveredDateParts{Day: true, Month: true, Time: true},
			formatString: "Monday, 2 January om 15u04",
			language:     "en_US",
		},
		{
			input:        "Sunday, 5 march om 21u01",
			coveredParts: CoveredDateParts{Day: true, Month: true, Time: true},
			formatString: "Monday, 2 January om 15u04",
			language:     "en_US",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat5(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "17-03-2023 20:30",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
			formatString: "2-1-2006 15:04",
		},
		{
			input:        "25-03-2023 20:30",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
			formatString: "2-1-2006 15:04",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat6(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "29 April",
			coveredParts: CoveredDateParts{Day: true, Month: true},
			formatString: "2 January",
			language:     "en_US",
		},
		{
			input:        "2 September",
			coveredParts: CoveredDateParts{Day: true, Month: true},
			formatString: "2 January",
			language:     "en_US",
		},
		{
			input:        "12 May",
			coveredParts: CoveredDateParts{Day: true, Month: true},
			formatString: "2 January",
			language:     "en_US",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat7(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "Fr. 17. Mär. 2023",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
			formatString: "Mon. 2. Jan. 2006",
			language:     "de_DE",
		},
		{
			input:        "Sa. 18. Mär. 2023",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
			formatString: "Mon. 2. Jan. 2006",
			language:     "de_DE",
		},
		{
			input:        "Fr. 24. Mär. 2023",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
			formatString: "Mon. 2. Jan. 2006",
			language:     "de_DE",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat8(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "Samedi 18 mars 2023",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
			formatString: "Monday 2 January 2006",
			language:     "fr_FR",
		},
		{
			input:        "Vendredi 24 mars 2023",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
			formatString: "Monday 2 January 2006",
			language:     "fr_FR",
		},
		{
			input:        "Samedi 25 mars 2023",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
			formatString: "Monday 2 January 2006",
			language:     "fr_FR",
		},
		{
			input:        "Dimanche 26 mars 2023",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
			formatString: "Monday 2 January 2006",
			language:     "fr_FR",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat9(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "ab 23 Uhr",
			coveredParts: CoveredDateParts{Time: true},
			formatString: "ab 15 Uhr",
		},
		{
			input:        "ab 21 Uhr",
			coveredParts: CoveredDateParts{Time: true},
			formatString: "ab 15 Uhr",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat10(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "March 17 @ 5:00 pm",
			coveredParts: CoveredDateParts{Day: true, Month: true, Time: true},
			formatString: "January 2 @ 15:04 pm",
			language:     "en_US",
		},
		{
			input:        "March 18 @ 11:30 pm",
			coveredParts: CoveredDateParts{Day: true, Month: true, Time: true},
			formatString: "January 2 @ 15:04 pm",
			language:     "en_US",
		},
		{
			input:        "April 1 @ 8:00 pm",
			coveredParts: CoveredDateParts{Day: true, Month: true, Time: true},
			formatString: "January 2 @ 15:04 pm",
			language:     "en_US",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat11(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "2.1.2012 Beginn: 15:04 Uhr",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
			formatString: "2.1.2006 Beginn: 15:04 Uhr",
		},
		{
			input:        "30.11.2022 Beginn: 11:30 Uhr",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
			formatString: "2.1.2006 Beginn: 15:04 Uhr",
		},
		{
			input:        "2.5.1994 Beginn: 6:13 Uhr",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
			formatString: "2.1.2006 Beginn: 15:04 Uhr",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat12(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "20:00h",
			coveredParts: CoveredDateParts{Time: true},
			formatString: "15:04h",
		},
		{
			input:        "23:30h",
			coveredParts: CoveredDateParts{Time: true},
			formatString: "15:04h",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat13(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "do 23 maart 2023",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
			formatString: "Mon 2 January 2006",
			language:     "nl_BE",
		},
		{
			input:        "wo 5 april 2023",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
			formatString: "Mon 2 January 2006",
			language:     "nl_BE",
		},
		{
			input:        "za 22 april 2023",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
			formatString: "Mon 2 January 2006",
			language:     "nl_BE",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat14(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "7.30pm",
			coveredParts: CoveredDateParts{Time: true},
			formatString: "15.04pm",
		},
		{
			input:        "9pm",
			coveredParts: CoveredDateParts{Time: true},
			formatString: "15pm",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat15(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "19:30 Uhr",
			coveredParts: CoveredDateParts{Time: true},
			formatString: "15:04 Uhr",
		},
		{
			input:        "20 Uhr",
			coveredParts: CoveredDateParts{Time: true},
			formatString: "15 Uhr",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat16(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "DOORS: 7:30PM",
			coveredParts: CoveredDateParts{Time: true},
			formatString: "DOORS: 15:04PM",
		},
		{
			input:        "DOORS: 5AM",
			coveredParts: CoveredDateParts{Time: true},
			formatString: "DOORS: 15PM",
		},
		{
			input:        "DOORS: 11:00AM",
			coveredParts: CoveredDateParts{Time: true},
			formatString: "DOORS: 15:04PM",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat17(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "nedeľa 25.02.2024 @18:00",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
			formatString: "Monday 2.1.2006 @15:04",
			language:     "sk_SK",
		},
		{
			input:        "piatok 01.03.2024 @20:00",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
			formatString: "Monday 2.1.2006 @15:04",
			language:     "sk_SK",
		},
		{
			input:        "štvrtok 07.03.2024 @18:30",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
			formatString: "Monday 2.1.2006 @15:04",
			language:     "sk_SK",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}

func TestGetDateFormat18(t *testing.T) {
	dateFormats := []formatTestStruct{
		{
			input:        "Mi. 04/12/2024",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
			formatString: "Mon. 2/1/2006",
			language:     "de_DE",
		},
		{
			input:        "Sa. 07/12/2024",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
			formatString: "Mon. 2/1/2006",
			language:     "de_DE",
		},
		{
			input:        "Sa. 18/01/2024",
			coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
			formatString: "Mon. 2/1/2006",
			language:     "de_DE",
		},
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}
