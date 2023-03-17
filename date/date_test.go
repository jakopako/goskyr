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
