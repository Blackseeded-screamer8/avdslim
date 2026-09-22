package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSlimImages(t *testing.T) {
	sdk := t.TempDir()
	abi := HostAbi()
	for _, rel := range []string{
		"android-34/google_apis/" + abi,
		"android-37.0/google_apis/" + abi,
		"android-37.1/google_apis_ps16k/" + abi,   // 16 KB: excluded
		"android-36/google_apis_playstore/" + abi, // Play Store: excluded
		"android-36/google_apis/other-abi",        // wrong arch: excluded
		"android-Baklava/google_apis/" + abi,      // codename: excluded
	} {
		if err := os.MkdirAll(filepath.Join(sdk, "system-images", rel), 0755); err != nil {
			t.Fatal(err)
		}
	}

	want := []string{
		"system-images;android-37.0;google_apis;" + abi,
		"system-images;android-34;google_apis;" + abi,
	}
	if got := SlimImages(sdk, 0); !reflect.DeepEqual(got, want) {
		t.Errorf("SlimImages(any) = %v, want %v", got, want)
	}
	if got := SlimImages(sdk, 34); !reflect.DeepEqual(got, want[1:]) {
		t.Errorf("SlimImages(34) = %v, want %v", got, want[1:])
	}
	if got := SlimImages(sdk, 36); len(got) != 0 {
		t.Errorf("SlimImages(36) = %v, want none (only playstore installed)", got)
	}
	if got := SlimImages(t.TempDir(), 0); len(got) != 0 {
		t.Errorf("empty SDK = %v, want none", got)
	}
}
