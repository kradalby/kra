package data

import (
	"embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed all:salary.yaml
var salary embed.FS

type (
	Salaries []Salary
	Salary   struct {
		Company   string `yaml:"company"`
		StartDate string `yaml:"start_date"`
		EndDate   string `yaml:"end_date"`
		HowILeft  string `yaml:"how_i_left"`
		Note      string `yaml:"note"`
		Salary    string `yaml:"salary"`
		Title     string `yaml:"title"`
	}
)

func ReadSalaries() (Salaries, error) {
	salaryByte, err := salary.ReadFile("salary.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read salaries: %w", err)
	}

	var salaries Salaries

	err = yaml.Unmarshal(salaryByte, &salaries)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal salaries: %w", err)
	}

	return salaries, nil
}
