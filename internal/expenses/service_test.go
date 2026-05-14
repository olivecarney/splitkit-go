package expenses

import "testing"

func TestParseMoney(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
	}{
		{name: "whole pounds", input: "12", want: 1200},
		{name: "pounds and pence", input: "12.34", want: 1234},
		{name: "single decimal place", input: "12.3", want: 1230},
		{name: "pound symbol", input: "£12.34", want: 1234},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMoney(tt.input)
			if err != nil {
				t.Fatalf("ParseMoney returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseMoney(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseMoneyRejectsInvalidAmounts(t *testing.T) {
	inputs := []string{"", "0", "-1", "12.345", "abc"}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseMoney(input); err == nil {
				t.Fatalf("ParseMoney(%q) expected error", input)
			}
		})
	}
}
