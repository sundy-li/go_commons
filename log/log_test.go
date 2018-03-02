package log

import "testing"

func TestLog(t *testing.T) {
	Debug("debug  fdsafds", 333)
	Info("critical  fdsaf", "33")
	Infof("critical %s fdsaf", "33")

	beeLogger.EnableFuncCallDepth(true)
	Critical("critical")
}
