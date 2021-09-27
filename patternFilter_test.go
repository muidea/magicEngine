package magicengine

import "testing"

func TestPatternFilter_Match(t *testing.T) {
	filter := NewPatternFilter("/api/v1/**")

	path := "/api/v1/access/abc/24323/"

	if !filter.Match(path) {
		t.Error("match failed")
		return
	}

}
