package unifi

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi"
)

// dev_id_override is the device type a client was manually set to. A client
// without an override reads as 0, the value that clears one.
func TestClientToModel_DevIdOverride(t *testing.T) {
	r := &clientResource{}
	v := int64(3263)

	for _, tc := range []struct {
		name string
		in   *int64
		want int64
	}{
		{"override", &v, 3263},
		{"none", nil, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &unifi.Client{MAC: "02:00:00:de:ad:03", DevIdOverride: tc.in}
			var model clientResourceModel
			if d := r.clientToModel(context.Background(), client, &model, "default"); d.HasError() {
				t.Fatalf("clientToModel: %v", d)
			}
			if model.DevIdOverride.IsNull() || model.DevIdOverride.ValueInt64() != tc.want {
				t.Errorf("dev_id_override = %v, want %d", model.DevIdOverride, tc.want)
			}
		})
	}
}

// The override is set through its own endpoint, only when the configured value
// differs from the controller's.
func TestDevIdOverrideChange(t *testing.T) {
	v := int64(3263)
	other := int64(4815)

	for _, tc := range []struct {
		name        string
		current     *int64
		planned     types.Int64
		wantValue   int64
		wantChanged bool
	}{
		{"unset in config", &v, types.Int64Unknown(), 0, false},
		{"null in config", &v, types.Int64Null(), 0, false},
		{"unchanged", &v, types.Int64Value(3263), 3263, false},
		{"changed", &v, types.Int64Value(4815), 4815, true},
		{"set from none", nil, types.Int64Value(4815), 4815, true},
		{"cleared", &other, types.Int64Value(0), 0, true},
		{"already clear", nil, types.Int64Value(0), 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, changed := devIdOverrideChange(tc.current, tc.planned)
			if changed != tc.wantChanged || value != tc.wantValue {
				t.Errorf("devIdOverrideChange = (%d, %t), want (%d, %t)",
					value, changed, tc.wantValue, tc.wantChanged)
			}
		})
	}
}

// The client PUT never carries dev_id_override: the controller sets it through
// the fingerprint endpoint, and echoing a stale value back could undo a change.
func TestMergeClient_DropsDevIdOverride(t *testing.T) {
	r := &clientResource{}
	v := int64(3263)
	merged := r.mergeClient(
		&unifi.Client{MAC: "02:00:00:de:ad:04", DevIdOverride: &v},
		&unifi.Client{MAC: "02:00:00:de:ad:04"},
	)
	if merged.DevIdOverride != nil {
		t.Errorf("DevIdOverride = %d, want nil", *merged.DevIdOverride)
	}
}
