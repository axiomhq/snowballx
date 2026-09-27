package stopwords

import "testing"

func TestListsLoad(t *testing.T) {
	want := []string{"danish", "dutch", "english", "finnish", "french", "german", "hungarian", "italian", "norwegian", "portuguese", "russian", "spanish", "swedish"}
	got := Languages()
	if len(got) != len(want) {
		t.Fatalf("languages %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("languages %v, want %v", got, want)
		}
	}
	if !Is("english", "the") || Is("english", "zzzz") || Is("greek", "και") {
		t.Fatal("membership")
	}
	if Set("greek") != nil {
		t.Fatal("greek has no list")
	}
	if n := len(Set("english")); n < 100 {
		t.Fatalf("english has %d words", n)
	}
}
