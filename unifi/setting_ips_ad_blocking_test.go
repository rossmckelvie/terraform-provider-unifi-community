package unifi

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/ubiquiti-community/go-unifi/unifi/settings"
)

// ipsModelManagingOnly returns an ips model that configures nothing except
// ad_blocking_network_ids.
func ipsModelManagingOnly(adBlocking types.List) *settingIpsModel {
	return &settingIpsModel{
		IPSMode:                             types.StringNull(),
		HoneypotEnabled:                     types.BoolNull(),
		ContentFilteringBlockingPageEnabled: types.BoolNull(),
		MemoryOptimized:                     types.BoolNull(),
		RestrictTorrents:                    types.BoolNull(),
		AdvancedFilteringPreference:         types.StringNull(),
		EnabledCategories:                   types.ListNull(types.StringType),
		EnabledNetworks:                     types.ListNull(types.StringType),
		AdBlockingNetworkIDs:                adBlocking,
		Honeypot: types.ListNull(
			types.ObjectType{AttrTypes: ipsHoneypotAttrTypes},
		),
		Suppression: types.ObjectNull(ipsSuppressionAttrTypes),
	}
}

func stringList(values ...string) types.List {
	elems := make([]attr.Value, len(values))
	for i, v := range values {
		elems[i] = types.StringValue(v)
	}
	return types.ListValueMust(types.StringType, elems)
}

func adBlockingNetworks(s *settings.Ips) []string {
	ids := []string{}
	for _, c := range s.AdBlockingConfigurations {
		ids = append(ids, c.NetworkID)
	}
	return ids
}

// ad_blocking_network_ids lists the networks ad blocking applies to. A list
// turns the site-wide switch on for exactly those networks; an empty list
// turns it off and leaves the stored networks alone, as the UI does; unset
// changes nothing.
func TestSettingIpsAdBlockingToSetting(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}
	on, off := true, false

	live := func() *settings.Ips {
		return &settings.Ips{
			AdBlockingEnabled: &on,
			AdBlockingConfigurations: []settings.SettingIpsAdBlockingConfigurations{
				{NetworkID: "kids"},
			},
		}
	}

	for _, tc := range []struct {
		name        string
		in          types.List
		wantEnabled *bool
		wantIDs     []string
	}{
		{"list sets networks", stringList("kids", "guest"), &on, []string{"kids", "guest"}},
		{"empty list turns off", stringList(), &off, []string{"kids"}},
		{"unset keeps controller", types.ListNull(types.StringType), &on, []string{"kids"}},
		{"unknown keeps controller", types.ListUnknown(types.StringType), &on, []string{"kids"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			got, _ := r.ipsModelToSetting(ctx, ipsModelManagingOnly(tc.in), &diags, live())
			if diags.HasError() {
				t.Fatalf("conversion: %v", diags)
			}
			if got.AdBlockingEnabled == nil || *got.AdBlockingEnabled != *tc.wantEnabled {
				t.Errorf(
					"ad_blocking_enabled = %v, want %t",
					got.AdBlockingEnabled,
					*tc.wantEnabled,
				)
			}
			if ids := adBlockingNetworks(got); !reflect.DeepEqual(ids, tc.wantIDs) {
				t.Errorf("ad_blocking_configurations = %v, want %v", ids, tc.wantIDs)
			}
		})
	}
}

// The controller's ad blocking is read back as the list of networks it is on
// for, and as an empty list when the switch is off, whatever networks are
// still stored.
func TestSettingIpsAdBlockingFromSetting(t *testing.T) {
	ctx := context.Background()
	r := &settingResource{}
	on, off := true, false
	configs := []settings.SettingIpsAdBlockingConfigurations{{NetworkID: "kids"}}

	for _, tc := range []struct {
		name    string
		setting *settings.Ips
		plan    types.List
		want    types.List
	}{
		{"on", &settings.Ips{AdBlockingEnabled: &on, AdBlockingConfigurations: configs}, stringList(), stringList("kids")},
		{"off", &settings.Ips{AdBlockingEnabled: &off, AdBlockingConfigurations: configs}, stringList("kids"), stringList()},
		{"absent", &settings.Ips{}, stringList(), stringList()},
		{"not configured", &settings.Ips{AdBlockingEnabled: &on, AdBlockingConfigurations: configs}, types.ListNull(types.StringType), types.ListNull(types.StringType)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			model := r.ipsSettingToModel(
				ctx,
				tc.setting,
				ipsModelManagingOnly(tc.plan),
				&diags,
				nil,
			)
			if diags.HasError() {
				t.Fatalf("conversion: %v", diags)
			}
			if !model.AdBlockingNetworkIDs.Equal(tc.want) {
				t.Errorf(
					"ad_blocking_network_ids = %v, want %v",
					model.AdBlockingNetworkIDs,
					tc.want,
				)
			}
		})
	}
}
