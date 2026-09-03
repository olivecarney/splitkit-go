package expenses

import (
	"reflect"
	"testing"

	"github.com/olivecarney/splitkit-go/internal/models"
)

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
	inputs := []string{"", "0", "-1", "12.345", "abc", "12,34", "1.2.3"}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseMoney(input); err == nil {
				t.Fatalf("ParseMoney(%q) expected error", input)
			}
		})
	}
}

func TestEqualSplitsRoundsRemainderToFirstParticipants(t *testing.T) {
	got, err := EqualSplits(1000, []string{"alice", "bob", "charlie"})
	if err != nil {
		t.Fatalf("EqualSplits returned error: %v", err)
	}

	want := []models.ExpenseSplit{
		{UserID: "alice", AmountCents: 334},
		{UserID: "bob", AmountCents: 333},
		{UserID: "charlie", AmountCents: 333},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("EqualSplits() = %#v, want %#v", got, want)
	}

	var total int64
	for _, split := range got {
		total += split.AmountCents
	}
	if total != 1000 {
		t.Fatalf("split total = %d, want 1000", total)
	}
}

func TestEqualSplitsRejectsInvalidParticipants(t *testing.T) {
	tests := []struct {
		name         string
		participants []string
	}{
		{name: "none", participants: nil},
		{name: "blank", participants: []string{"alice", ""}},
		{name: "duplicate", participants: []string{"alice", "alice"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := EqualSplits(1000, tt.participants); err == nil {
				t.Fatal("EqualSplits expected error")
			}
		})
	}
}
