package aemet

import (
	"testing"
)

func TestGetMaritimeForecastMapGIF(t *testing.T) {
	_, err := GetMaritimeForecastMapGIF("ecwam_bal", 24, 150, true)

	if err != nil {
		t.Errorf("Unexpected error getting GIF: %v", err)
	}
}
