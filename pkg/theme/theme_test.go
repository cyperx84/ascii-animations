package theme

import (
	"testing"
)

func TestPaletteColors(t *testing.T) {
	colors := []struct {
		name  string
		color interface{}
	}{
		{"BgColor", BgColor},
		{"FgColor", FgColor},
		{"CyanColor", CyanColor},
		{"GreenColor", GreenColor},
		{"PurpleColor", PurpleColor},
		{"RedColor", RedColor},
	}
	for _, c := range colors {
		if c.color == nil {
			t.Errorf("%s is nil", c.name)
		}
	}
}

func TestStylesRender(t *testing.T) {
	// verify styles can render without panic
	styles := []struct {
		name   string
		render string
	}{
		{"Title", Title.Render("test")},
		{"MenuItem", MenuItem.Render("test")},
		{"SelectedItem", SelectedItem.Render("test")},
		{"Header", Header.Render("test")},
		{"Footer", Footer.Render("test")},
		{"Help", Help.Render("test")},
		{"AnimName", AnimName.Render("test")},
		{"AnimDesc", AnimDesc.Render("test")},
		{"Lib", Lib.Render("test")},
		{"Source", Source.Render("test")},
		{"ExportMsg", ExportMsg.Render("test")},
		{"Preview", Preview.Render("test")},
		{"InputStyle", InputStyle.Render("test")},
	}
	for _, s := range styles {
		if s.render == "" {
			t.Errorf("style %s rendered empty string", s.name)
		}
	}
}
