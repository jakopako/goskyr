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

func TestGetDateFormat(t *testing.T) {
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
		// {
		// 	input:        "Wednesday, 1 march om 21u00",
		// 	coveredParts: CoveredDateParts{Day: true, Month: true, Time: true},
		// 	formatString: "Monday, 2 january om 15u04",
		// 	language:     "en_US",
		// },
		// {
		// 	input:        "Thursday, 2 march om 21u00",
		// 	coveredParts: CoveredDateParts{Day: true, Month: true, Time: true},
		// 	formatString: "Monday, 2 january om 15u04",
		// 	language:     "en_US",
		// },
		// {
		// 	input:        "Sunday, 5 march om 21u01",
		// 	coveredParts: CoveredDateParts{Day: true, Month: true, Time: true},
		// 	formatString: "Monday, 2 january om 15u04",
		// 	language:     "en_US",
		// },
		// {
		// 	input:        "17-03-2023 20:30",
		// 	coveredParts: CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
		// 	formatString: "2-1-2006 15:04",
		// },
		// {
		// 	input:        "25-03-2023 20:30",
		// 	coveredParts: CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
		// 	formatString: "2-1-2006 15:04",
		// },
		// {
		// 	input:        []string{"29 April"},
		// 	coveredParts: CoveredDateParts{Day: true, Month: true},
		// 	formatString: "2 January",
		// 	language:     "en_US",
		// },
		// {
		// 	input:        []string{"Fr. 17. Mär. 2023", "Sa. 18. Mär. 2023", "Fr. 24. Mär. 2023"},
		// 	coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
		// 	formatString: "Mon. 2. Jan. 2006",
		// 	language:     "de_DE",
		// },
		// {
		// 	input:        []string{"Samedi 18 mars 2023", "Vendredi 24 mars 2023", "Samedi 25 mars 2023", "Dimanche 26 mars 2023"},
		// 	coveredParts: CoveredDateParts{Day: true, Month: true, Year: true},
		// 	formatString: "Mon. 2. Jan. 2006",
		// 	language:     "de_DE",
		// },
		// {
		// 	input:        []string{"ab 23 Uhr", "ab 21 Uhr"},
		// 	coveredParts: CoveredDateParts{Time: true},
		// 	formatString: "ab 15 Uhr",
		// },
	}
	for _, df := range dateFormats {
		f, l := GetDateFormat(df.input, &df.coveredParts)
		if f != df.formatString {
			log.Fatalf("expected %s but got %s", df.formatString, f)
		}
		if l != df.language {
			log.Fatalf("expected date language %s but got %s", df.language, l)
		}
	}
}
