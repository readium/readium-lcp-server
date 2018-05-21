package model_test

import "testing"

func TestDashboardStore_GetDashboardInfos(t *testing.T) {
	info, err := stor.Dashboard().GetDashboardInfos()
	if err != nil {
		t.Fatalf("Error : %v", err)
	}
	t.Logf("Dashboard info : %#v", info)
}

func TestDashboardStore_GetDashboardBestSellers(t *testing.T) {
	info, err := stor.Dashboard().GetDashboardBestSellers()
	if err != nil {
		t.Fatalf("Error : %v", err)
	}
	for _, bs := range info {
		t.Logf("Dashboard best seller : %#v", bs)
	}
}
