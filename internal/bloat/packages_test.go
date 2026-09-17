package bloat

import "testing"

func TestNoBootCriticalPackages(t *testing.T) {
	all := append([]string{}, AggressiveBloatPackages...)
	for _, pkgs := range StandardBloatCategories {
		all = append(all, pkgs...)
	}
	for _, p := range all {
		for _, c := range BootCritical {
			if p == c {
				t.Errorf("bloat list contains boot-critical package %s", c)
			}
		}
	}
}
