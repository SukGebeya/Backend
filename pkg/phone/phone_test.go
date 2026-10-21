package phone

import "testing"

func TestNormalize(t *testing.T) {
	valid := map[string]string{
		"0911234567":       "+251911234567",
		"911234567":        "+251911234567",
		"251911234567":     "+251911234567",
		"+251911234567":    "+251911234567",
		"+251 91 123 4567": "+251911234567",
		"00251911234567":   "+251911234567",
		"091-123-4567":     "+251911234567",
		"0711234567":       "+251711234567",
		"+251711234567":    "+251711234567",
	}
	for in, want := range valid {
		got, err := Normalize(in)
		if err != nil || got != want {
			t.Errorf("Normalize(%q) = %q, %v; want %q", in, got, err, want)
		}
	}

	invalid := []string{"", "0111234567", "+1234567890", "09112345", "09112345678", "+251811234567", "abc0911234567", "0911+234567"}
	for _, in := range invalid {
		if got, err := Normalize(in); err == nil {
			t.Errorf("Normalize(%q) = %q; want error", in, got)
		}
	}
}

func TestLocal(t *testing.T) {
	got, err := Local("+251911234567")
	if err != nil || got != "0911234567" {
		t.Errorf("Local = %q, %v", got, err)
	}
	got, err = Local("711234567")
	if err != nil || got != "0711234567" {
		t.Errorf("Local = %q, %v", got, err)
	}
}

func TestNormalizePtr(t *testing.T) {
	if got, err := NormalizePtr(nil); got != nil || err != nil {
		t.Errorf("nil should stay nil")
	}
	empty := "  "
	if got, err := NormalizePtr(&empty); got != nil || err != nil {
		t.Errorf("blank should become nil")
	}
	raw := "0911234567"
	got, err := NormalizePtr(&raw)
	if err != nil || *got != "+251911234567" {
		t.Errorf("NormalizePtr = %v, %v", got, err)
	}
}
