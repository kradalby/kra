package data

import (
	"testing"
)

func TestReadSalaries(t *testing.T) {
	salaries, err := ReadSalaries()
	if err != nil {
		t.Fatalf("ReadSalaries() returned error: %v", err)
	}

	if len(salaries) == 0 {
		t.Fatal("ReadSalaries() returned empty slice")
	}

	// Verify that all required fields are populated for each entry
	for i, s := range salaries {
		if s.Title == "" {
			t.Errorf("salary[%d]: Title is empty", i)
		}
		if s.StartDate == "" {
			t.Errorf("salary[%d]: StartDate is empty", i)
		}
		if s.EndDate == "" {
			t.Errorf("salary[%d]: EndDate is empty", i)
		}
		if s.Salary == "" {
			t.Errorf("salary[%d]: Salary is empty", i)
		}
		if s.HowILeft == "" {
			t.Errorf("salary[%d]: HowILeft is empty", i)
		}
	}
}

func TestReadSalariesFieldCount(t *testing.T) {
	salaries, err := ReadSalaries()
	if err != nil {
		t.Fatalf("ReadSalaries() returned error: %v", err)
	}

	// The salary.yaml currently has 9 entries
	const expectedCount = 9
	if len(salaries) != expectedCount {
		t.Errorf("ReadSalaries() returned %d entries, want %d", len(salaries), expectedCount)
	}
}

func TestReadSalariesFirstEntry(t *testing.T) {
	salaries, err := ReadSalaries()
	if err != nil {
		t.Fatalf("ReadSalaries() returned error: %v", err)
	}

	first := salaries[0]
	if first.Title != "Member of Technical Staff" {
		t.Errorf("first salary Title = %q, want %q", first.Title, "Member of Technical Staff")
	}
	if first.EndDate != "current" {
		t.Errorf("first salary EndDate = %q, want %q", first.EndDate, "current")
	}
}

func TestReadSalariesLastEntry(t *testing.T) {
	salaries, err := ReadSalaries()
	if err != nil {
		t.Fatalf("ReadSalaries() returned error: %v", err)
	}

	last := salaries[len(salaries)-1]
	if last.Title != "Summer job" {
		t.Errorf("last salary Title = %q, want %q", last.Title, "Summer job")
	}
	if last.Salary != "NOK 185/hour" {
		t.Errorf("last salary Salary = %q, want %q", last.Salary, "NOK 185/hour")
	}
}
