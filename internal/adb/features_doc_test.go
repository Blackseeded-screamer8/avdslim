package adb

import (
	"os"
	"strings"
	"testing"
)

// featureDoc is the user-facing reference `enable`/`disable` targets are listed in.
const featureDoc = "../../docs/FEATURES.md"

// Every alias and package a user can type must appear in the reference, or the
// doc has drifted from the code.
func TestFeatureDocListsAliases(t *testing.T) {
	data, err := os.ReadFile(featureDoc)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(data)

	for alias, pkgs := range featureAliases {
		if !strings.Contains(doc, "`"+alias+"`") {
			t.Errorf("%s does not list alias %q", featureDoc, alias)
		}
		for _, p := range pkgs {
			if !strings.Contains(doc, p) {
				t.Errorf("%s does not list package %q (alias %q)", featureDoc, p, alias)
			}
		}
	}

	for _, g := range SkipGroups {
		if !strings.Contains(doc, "`"+g+"`") {
			t.Errorf("%s does not list --skip group %q", featureDoc, g)
		}
	}
}
