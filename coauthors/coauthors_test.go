package coauthors

import (
	"reflect"
	"testing"
)

func TestCreateCommitMessage(t *testing.T) {
	expected := `

# automatically added all co-authors from WIP commits
# add missing co-authors manually
Co-authored-by: Alice <alice@example.com>
Co-authored-by: Bob <bob@example.com>
`
	actual := createCommitMessage([]Author{"Alice <alice@example.com>", "Bob <bob@example.com>"})
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

func TestSortByLength(t *testing.T) {
	slice := []string{"aa", "b"}

	sortByLength(slice)

	expected := []string{"b", "aa"}
	if !reflect.DeepEqual(expected, slice) {
		t.Errorf("expected %v, got %v", expected, slice)
	}
}

func TestRemoveDuplicateValues(t *testing.T) {
	slice := []string{"aa", "b", "c", "b"}

	actual := removeDuplicateValues(slice)

	expected := []string{"aa", "b", "c"}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}
