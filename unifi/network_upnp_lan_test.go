package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-nettypes/cidrtypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// upnp_lan_enabled is the per-network UPnP opt-in. A known value is sent; an
// unknown or null one sends false, which on update is diffed away against the
// controller's value because the attribute uses the prior state when unset.
func TestModelToNetworkUPnPLanEnabled(t *testing.T) {
	ctx := context.Background()
	r := &networkResource{}

	for _, tc := range []struct {
		name string
		in   types.Bool
		want bool
	}{
		{"true", types.BoolValue(true), true},
		{"false", types.BoolValue(false), false},
		{"null", types.BoolNull(), false},
		{"unknown", types.BoolUnknown(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			model := networkResourceModel{
				Name:           types.StringValue("gaming"),
				Subnet:         cidrtypes.NewIPv4PrefixValue("192.168.45.1/24"),
				UPnPLanEnabled: tc.in,
			}
			api, diags := r.modelToNetwork(ctx, &model)
			if diags.HasError() {
				t.Fatalf("modelToNetwork: %v", diags)
			}
			if api.UPnPLanEnabled != tc.want {
				t.Errorf("UPnPLanEnabled = %t, want %t", api.UPnPLanEnabled, tc.want)
			}
		})
	}
}

// The controller's upnp_lan_enabled is read back, so an adopted network shows
// it and a change made in the UI is visible as drift.
func TestNetworkToModelUPnPLanEnabled(t *testing.T) {
	ctx := context.Background()
	r := &networkResource{}

	for _, want := range []bool{true, false} {
		network := &unifi.Network{
			ID:             "n1",
			Name:           new("gaming"),
			Purpose:        unifi.PurposeCorporate,
			IPSubnet:       new("192.168.45.1/24"),
			UPnPLanEnabled: want,
		}
		var model networkResourceModel
		if d := r.networkToModel(ctx, network, &model, "default", nil); d.HasError() {
			t.Fatalf("networkToModel: %v", d)
		}
		if got := model.UPnPLanEnabled; got.IsNull() || got.ValueBool() != want {
			t.Errorf("UPnPLanEnabled = %v, want %t", got, want)
		}
	}
}
