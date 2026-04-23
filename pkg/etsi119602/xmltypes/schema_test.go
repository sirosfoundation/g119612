package xmltypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindByLanguage(t *testing.T) {
	en := Lang("en")
	sv := Lang("sv")
	enVal := NonEmptyNormalizedString("English")
	svVal := NonEmptyNormalizedString("Svenska")

	names := &InternationalNamesType{
		Name: []*MultiLangNormStringType{
			{XmlLangAttr: &en, NonEmptyNormalizedString: &enVal},
			{XmlLangAttr: &sv, NonEmptyNormalizedString: &svVal},
		},
	}

	assert.Equal(t, "English", FindByLanguage(names, "en", ""))
	assert.Equal(t, "Svenska", FindByLanguage(names, "sv", ""))
	assert.Equal(t, "default", FindByLanguage(names, "de", "default"))
	assert.Equal(t, "dflt", FindByLanguage(nil, "en", "dflt"))
}
