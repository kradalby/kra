package html

import (
	"strings"
	"testing"

	"github.com/chasefleming/elem-go"
	"github.com/kradalby/kra/data"
)

func TestGravatar(t *testing.T) {
	img := Gravatar(200)
	rendered := img.Render()

	if !strings.Contains(rendered, "gravatar.com/avatar/") {
		t.Error("Gravatar should contain gravatar.com URL")
	}
	if !strings.Contains(rendered, "s=200") {
		t.Error("Gravatar should contain size parameter")
	}
	if !strings.Contains(rendered, "<img") {
		t.Error("Gravatar should render an img tag")
	}
}

func TestAlink(t *testing.T) {
	link := Alink("https://example.com", "Example")
	rendered := link.Render()

	if !strings.Contains(rendered, `href="https://example.com"`) {
		t.Errorf("Alink should contain href, got: %s", rendered)
	}
	if !strings.Contains(rendered, "Example") {
		t.Errorf("Alink should contain text, got: %s", rendered)
	}
	if !strings.Contains(rendered, "<a") {
		t.Errorf("Alink should render an anchor tag, got: %s", rendered)
	}
}

func TestLinkElem(t *testing.T) {
	inner := Gravatar(48)
	link := LinkElem("https://example.com", inner)
	rendered := link.Render()

	if !strings.Contains(rendered, `href="https://example.com"`) {
		t.Errorf("LinkElem should contain href, got: %s", rendered)
	}
	if !strings.Contains(rendered, "<img") {
		t.Errorf("LinkElem should contain the inner element, got: %s", rendered)
	}
}

func TestSalaryRowsEmpty(t *testing.T) {
	rows := SalaryRows(data.Salaries{})
	if len(rows) != 0 {
		t.Errorf("SalaryRows with empty input should return 0 rows, got %d", len(rows))
	}
}

func TestSalaryRowsPopulated(t *testing.T) {
	salaries := data.Salaries{
		{
			Title:     "Engineer",
			StartDate: "2020-01-01",
			EndDate:   "2021-01-01",
			HowILeft:  "quit",
			Salary:    "EUR 100000",
			Note:      "test note",
		},
		{
			Title:     "Senior Engineer",
			StartDate: "2021-01-01",
			EndDate:   "current",
			HowILeft:  "N/A",
			Salary:    "EUR 150000",
			Note:      "",
		},
	}

	rows := SalaryRows(salaries)
	if len(rows) != 2 {
		t.Errorf("SalaryRows should return 2 rows, got %d", len(rows))
	}

	rendered := rows[0].Render()
	if !strings.Contains(rendered, "Engineer") {
		t.Errorf("first row should contain title, got: %s", rendered)
	}
	if !strings.Contains(rendered, "EUR 100000") {
		t.Errorf("first row should contain salary, got: %s", rendered)
	}
	if !strings.Contains(rendered, "<tr") {
		t.Errorf("row should be a table row, got: %s", rendered)
	}
}

func TestHomePage(t *testing.T) {
	page := Home()
	rendered := page.Render()

	checks := []string{
		"<html",
		"kradalby.no",
		"Tailscale",
		"Headscale",
		"github.com/kradalby",
		"linkedin.com/in/kradalby",
	}

	for _, check := range checks {
		if !strings.Contains(rendered, check) {
			t.Errorf("Home page should contain %q", check)
		}
	}
}

func TestAboutPage(t *testing.T) {
	page := About()
	rendered := page.Render()

	checks := []string{
		"<html",
		"About me",
		"Kristoffer",
		"Tailscale",
		"Public speaking",
		"FOSDEM",
	}

	for _, check := range checks {
		if !strings.Contains(rendered, check) {
			t.Errorf("About page should contain %q", check)
		}
	}
}

func TestSalaryPage(t *testing.T) {
	page, err := Salary()
	if err != nil {
		t.Fatalf("Salary() returned error: %v", err)
	}

	rendered := page.Render()

	checks := []string{
		"<html",
		"Salary transparency",
		"<table",
		"<thead",
		"<tbody",
		"Title",
		"Start date",
		"End date",
	}

	for _, check := range checks {
		if !strings.Contains(rendered, check) {
			t.Errorf("Salary page should contain %q", check)
		}
	}
}

func TestBootstrap(t *testing.T) {
	page := Bootstrap(PageMeta{Title: "Test Page"}, nil)
	rendered := page.Render()

	if !strings.Contains(rendered, `lang="en"`) {
		t.Error("Bootstrap should set lang=en")
	}
	if !strings.Contains(rendered, "Test Page - kradalby.no") {
		t.Error("Bootstrap should format title correctly")
	}
	if !strings.Contains(rendered, "charset") {
		t.Error("Bootstrap should include charset meta")
	}
	if !strings.Contains(rendered, "viewport") {
		t.Error("Bootstrap should include viewport meta")
	}
}

func TestBootstrapEmptyTitle(t *testing.T) {
	page := Bootstrap(PageMeta{}, nil)
	rendered := page.Render()

	if !strings.Contains(rendered, "kradalby.no") {
		t.Error("Bootstrap with empty title should still show site name")
	}
	if strings.Contains(rendered, " - kradalby.no") {
		t.Error("Bootstrap with empty title should not have ' - ' prefix")
	}
}

func TestPageBase(t *testing.T) {
	page := PageBase(PageMeta{Title: "Test"})
	rendered := page.Render()

	if !strings.Contains(rendered, "<nav") {
		t.Error("PageBase should include nav")
	}
	if !strings.Contains(rendered, "<main") {
		t.Error("PageBase should include main")
	}
	if !strings.Contains(rendered, "<footer") {
		t.Error("PageBase should include footer")
	}
	if !strings.Contains(rendered, "Copyright") {
		t.Error("PageBase should include copyright")
	}
	if !strings.Contains(rendered, `href="/"`) {
		t.Error("PageBase should include home link")
	}
	if !strings.Contains(rendered, `href="/about"`) {
		t.Error("PageBase should include about link")
	}
}

func TestSvgIcon(t *testing.T) {
	tests := []struct {
		name string
		fn   func(int) elem.Node
	}{
		{"Instagram", Instagram},
		{"Linkedin", Linkedin},
		{"Github", Github},
		{"Discord", Discord},
		{"Twitter", Twitter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := tt.fn(24)
			rendered := icon.Render()

			if !strings.Contains(rendered, "<svg") {
				t.Errorf("%s should render an SVG element", tt.name)
			}
			if !strings.Contains(rendered, "<path") {
				t.Errorf("%s should contain a path element", tt.name)
			}
			if !strings.Contains(rendered, `height="24"`) {
				t.Errorf("%s should have height=24", tt.name)
			}
			if !strings.Contains(rendered, `width="24"`) {
				t.Errorf("%s should have width=24", tt.name)
			}
		})
	}
}
