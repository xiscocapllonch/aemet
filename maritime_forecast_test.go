package aemet

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestFormatDate(t *testing.T) {
	expectedOut := "domingo, 7 junio 2020 a las 20:00 hora oficial"

	date := Date("2020-06-07T20:00:00")
	formattedDate := date.formatDate()

	if formattedDate != expectedOut {
		t.Errorf("Expected Output: %v\nBut got: %v", expectedOut, formattedDate)
	}
}

func TestGetXML(t *testing.T) {
	expectedWarning := "Posibilidad de aguaceros y tormentas muy fuertes."

	result, err := getMockXML(t)

	if err != nil {
		t.Errorf("error getting XML: %v", err)
	} else {
		if result.Warning.Text != expectedWarning {
			t.Errorf(
				"Warning Text should be: %v\nBut got: %v",
				expectedWarning,
				result.Warning.Text,
			)
		}
	}
}

func TestGetMaritimeForecast(t *testing.T) {
	forecastText, err := GetMaritimeForecast("FQXX44MM")

	if err != nil {
		t.Errorf("error getting XML: %v", err)
	} else {

		testTxt := []string{
			"<u><b>Situación General Illes Balears</b></u>",
			"<b>Noroeste de Mallorca (de Dragonera a Formentor)</b>",
			"<b>Aguas costeras de Formentera</b>",
		}

		for _, substr := range testTxt {
			if !strings.Contains(forecastText, substr) {
				t.Errorf(
					"Expect text contains: %v\n\n\nBut got: %v",
					substr,
					forecastText,
				)
			}
		}
	}
}

func getMockXML(t *testing.T) (Result, error) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		file, err := os.Open("test_data/test.xml")
		if err != nil {
			t.Errorf("error opening test file: %v", err)
		}

		defer func() {
			err = file.Close()
		}()

		if err != nil {
			t.Errorf("error closing test file: %v", err)
		}

		bFile, err := ioutil.ReadAll(file)

		_, err = w.Write(bFile)
		if err != nil {
			t.Errorf("error writing test file: %v", err)
		}
	}))

	defer ts.Close()

	return getXML(ts.URL)
}
