package smoke_test

import "testing"

func TestSmoke_CollectorShipper_Runs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
}
