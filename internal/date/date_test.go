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

func TestCheckForDoubleDateParts(t *testing.T) {
	tests := []struct {
		name    string
		dpOne   CoveredDateParts
		dpTwo   CoveredDateParts
		wantErr string
	}{
		{
			name:    "No overlap",
			dpOne:   CoveredDateParts{Day: true},
			dpTwo:   CoveredDateParts{Month: true},
			wantErr: "",
		},
		{
			name:    "Day overlap",
			dpOne:   CoveredDateParts{Day: true},
			dpTwo:   CoveredDateParts{Day: true},
			wantErr: "date parsing error: 'day' covered at least twice",
		},
		{
			name:    "Month overlap",
			dpOne:   CoveredDateParts{Month: true},
			dpTwo:   CoveredDateParts{Month: true},
			wantErr: "date parsing error: 'month' covered at least twice",
		},
		{
			name:    "Year overlap",
			dpOne:   CoveredDateParts{Year: true},
			dpTwo:   CoveredDateParts{Year: true},
			wantErr: "date parsing error: 'year' covered at least twice",
		},
		{
			name:    "Time overlap",
			dpOne:   CoveredDateParts{Time: true},
			dpTwo:   CoveredDateParts{Time: true},
			wantErr: "date parsing error: 'time' covered at least twice",
		},
		{
			name:    "Multiple overlaps, only first detected",
			dpOne:   CoveredDateParts{Day: true, Month: true},
			dpTwo:   CoveredDateParts{Day: true, Month: true},
			wantErr: "date parsing error: 'day' covered at least twice",
		},
		{
			name:    "No overlap, all different",
			dpOne:   CoveredDateParts{Day: true, Month: true},
			dpTwo:   CoveredDateParts{Year: true, Time: true},
			wantErr: "",
		},
		{
			name:    "All overlap, only first detected",
			dpOne:   CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
			dpTwo:   CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
			wantErr: "date parsing error: 'day' covered at least twice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckForDoubleDateParts(tt.dpOne, tt.dpTwo)
			if tt.wantErr == "" && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.wantErr)
				} else if err.Error() != tt.wantErr {
					t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
				}
			}
		})
	}
}
func TestMergeDateParts(t *testing.T) {
	tests := []struct {
		name     string
		dpOne    CoveredDateParts
		dpTwo    CoveredDateParts
		expected CoveredDateParts
	}{
		{
			name:     "No overlap, all false",
			dpOne:    CoveredDateParts{},
			dpTwo:    CoveredDateParts{},
			expected: CoveredDateParts{},
		},
		{
			name:     "No overlap, different parts",
			dpOne:    CoveredDateParts{Day: true},
			dpTwo:    CoveredDateParts{Month: true},
			expected: CoveredDateParts{Day: true, Month: true},
		},
		{
			name:     "Overlap, both true for Day",
			dpOne:    CoveredDateParts{Day: true},
			dpTwo:    CoveredDateParts{Day: true},
			expected: CoveredDateParts{Day: true},
		},
		{
			name:     "All true in one, all false in other",
			dpOne:    CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
			dpTwo:    CoveredDateParts{},
			expected: CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
		},
		{
			name:     "All true in both",
			dpOne:    CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
			dpTwo:    CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
			expected: CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
		},
		{
			name:     "Mixed overlap",
			dpOne:    CoveredDateParts{Day: true, Year: true},
			dpTwo:    CoveredDateParts{Month: true, Time: true},
			expected: CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
		},
		{
			name:     "Partial overlap",
			dpOne:    CoveredDateParts{Day: true, Month: true},
			dpTwo:    CoveredDateParts{Month: true, Year: true},
			expected: CoveredDateParts{Day: true, Month: true, Year: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeDateParts(tt.dpOne, tt.dpTwo)
			if result != tt.expected {
				t.Errorf("expected %+v, got %+v", tt.expected, result)
			}
		})
	}
}

func TestHasAllDateParts(t *testing.T) {
	tests := []struct {
		name     string
		cdp      CoveredDateParts
		expected bool
	}{
		{
			name:     "All parts true",
			cdp:      CoveredDateParts{Day: true, Month: true, Year: true, Time: true},
			expected: true,
		},
		{
			name:     "Day missing",
			cdp:      CoveredDateParts{Day: false, Month: true, Year: true, Time: true},
			expected: false,
		},
		{
			name:     "Month missing",
			cdp:      CoveredDateParts{Day: true, Month: false, Year: true, Time: true},
			expected: false,
		},
		{
			name:     "Year missing",
			cdp:      CoveredDateParts{Day: true, Month: true, Year: false, Time: true},
			expected: false,
		},
		{
			name:     "Time missing",
			cdp:      CoveredDateParts{Day: true, Month: true, Year: true, Time: false},
			expected: false,
		},
		{
			name:     "All parts false",
			cdp:      CoveredDateParts{Day: false, Month: false, Year: false, Time: false},
			expected: false,
		},
		{
			name:     "Only one part true",
			cdp:      CoveredDateParts{Day: true},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasAllDateParts(tt.cdp)
			if result != tt.expected {
				t.Errorf("HasAllDateParts(%+v) = %v; want %v", tt.cdp, result, tt.expected)
			}
		})
	}
}
