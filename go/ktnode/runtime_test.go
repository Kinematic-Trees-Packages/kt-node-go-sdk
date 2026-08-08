package ktnode

import "testing"

func TestABIVersionLoadsFromRuntime(t *testing.T) {
	major, minor := ABIVersion()
	if major != uint32(ABIMajor) || minor != uint32(ABIMinor) {
		t.Fatalf("ABI version=%d.%d constants=%d.%d", major, minor, ABIMajor, ABIMinor)
	}
	if BuildID() == "" {
		t.Fatal("expected non-empty runtime build id")
	}
}

func TestNodeFuncDefaultsAndPanicSafety(t *testing.T) {
	node := NodeFunc{}
	if node.Setup(nil) != Continue || node.Step(nil) != Stop || node.Close(nil) != Stop {
		t.Fatal("unexpected default outcomes")
	}
}
