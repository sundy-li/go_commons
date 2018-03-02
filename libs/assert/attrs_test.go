package assert

import (
	"bytes"
	"fmt"
	"testing"
)

func TestShellAttributes(t *testing.T) {
	var buffer bytes.Buffer

	output := NewWriter(&buffer, Underline, Blue)
	output.Write([]byte("testing"))
	if buffer.String() != fmt.Sprintf("%c[0;4;34mtesting%c[0m", 27, 27) {
		t.Errorf("got wrong output somehow")
	}
}
