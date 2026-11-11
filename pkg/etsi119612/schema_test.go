package etsi119612

import "testing"

func TestFindByLanguage(t *testing.T) {
	val := NonEmptyNormalizedString("Operator")
	other := NonEmptyNormalizedString("Betreiber")
	en, de := Lang("en"), Lang("de")

	tests := []struct {
		name  string
		names *InternationalNamesType
		want  string
	}{
		{"nil names", nil, "Unknown"},
		{"empty names", &InternationalNamesType{}, "Unknown"},
		{"nil entry", &InternationalNamesType{Name: []*MultiLangNormStringType{nil}}, "Unknown"},
		{"missing xml:lang", &InternationalNamesType{Name: []*MultiLangNormStringType{{NonEmptyNormalizedString: &val}}}, "Unknown"},
		{"missing value", &InternationalNamesType{Name: []*MultiLangNormStringType{{XmlLangAttr: &en}}}, "Unknown"},
		{"match after unlabelled entry", &InternationalNamesType{Name: []*MultiLangNormStringType{
			{NonEmptyNormalizedString: &other},
			{XmlLangAttr: &de, NonEmptyNormalizedString: &other},
			{XmlLangAttr: &en, NonEmptyNormalizedString: &val},
		}}, "Operator"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FindByLanguage(tt.names, "en", "Unknown"); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
