package config

import (
	"context"
	"encoding/json"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

func TestDefaultFailoverOptions(t *testing.T) {
	o := DefaultHiddifyOptions()
	if o.Tolerance != 150 || o.MinDwell != 60 || o.StallTimeout != 8 || o.StallThreshold != 3 ||
		o.StallWindow != 30 || o.RescueBatch != 6 || o.RescueTimeout != 5 || o.ActiveCheckInterval != 180 {
		t.Fatalf("defaults: %+v", o.FailoverOptions)
	}
	if !o.DisableInterfaceSweep || o.URLTestInterval != DurationInSeconds(1800) {
		t.Fatalf("sweep defaults: disable=%v interval=%v", o.DisableInterfaceSweep, o.URLTestInterval)
	}
}

func TestFailoverOptionsUnmarshalFromFlatJSON(t *testing.T) {
	o := DefaultHiddifyOptions()
	flat := []byte(`{"failover-tolerance":222,"failover-rescue-batch":4,"disable-interface-sweep":false}`)
	if err := json.Unmarshal(flat, o); err != nil {
		t.Fatal(err)
	}
	if o.Tolerance != 222 {
		t.Fatalf("Tolerance: got %d, want 222", o.Tolerance)
	}
	if o.RescueBatch != 4 {
		t.Fatalf("RescueBatch: got %d, want 4", o.RescueBatch)
	}
	if o.DisableInterfaceSweep != false {
		t.Fatalf("DisableInterfaceSweep: got %v, want false", o.DisableInterfaceSweep)
	}
	if o.MinDwell != 60 {
		t.Fatalf("MinDwell should keep its default: got %d, want 60", o.MinDwell)
	}

	b, err := json.Marshal(DefaultHiddifyOptions())
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["failover-tolerance"]; !ok {
		t.Fatalf("marshaled JSON must carry top-level key failover-tolerance: %s", b)
	}
	if _, ok := raw["Failover"]; ok {
		t.Fatalf("marshaled JSON must not nest failover fields under Failover: %s", b)
	}
}

func TestLowestBalancerCarriesFailoverOptions(t *testing.T) {
	opt := DefaultHiddifyOptions()
	input := option.Options{Outbounds: []option.Outbound{
		{Type: C.TypeDirect, Tag: "s1", Options: &option.DirectOutboundOptions{}},
		{Type: C.TypeDirect, Tag: "s2", Options: &option.DirectOutboundOptions{}},
	}}
	out, err := BuildConfig(context.Background(), opt, &ReadOptions{Options: &input})
	if err != nil {
		t.Fatal(err)
	}
	var lowest *option.BalancerOutboundOptions
	for _, ob := range out.Outbounds {
		if ob.Tag == OutboundURLTestTag {
			lowest = ob.Options.(*option.BalancerOutboundOptions)
		}
	}
	if lowest == nil {
		t.Fatal("lowest balancer missing")
	}
	if lowest.Tolerance != 150 || lowest.MinDwell.Build().Seconds() != 60 || lowest.RescueBatch != 6 || lowest.ActiveCheckInterval.Build().Minutes() != 3 {
		t.Fatalf("lowest options: %+v", lowest)
	}
	if out.Experimental == nil || out.Experimental.Monitoring == nil || !out.Experimental.Monitoring.DisableInterfaceSweep {
		t.Fatal("monitoring must carry disable_interface_sweep")
	}
}

// TestSelectorDefaultsToLowest pins R6: a fresh profile must start on the lowest-delay balancer,
// the only outbound that carries the failover controller, not on the round-robin one. Both stay
// selectable.
func TestSelectorDefaultsToLowest(t *testing.T) {
	opt := DefaultHiddifyOptions()
	input := option.Options{Outbounds: []option.Outbound{
		{Type: C.TypeDirect, Tag: "s1", Options: &option.DirectOutboundOptions{}},
		{Type: C.TypeDirect, Tag: "s2", Options: &option.DirectOutboundOptions{}},
	}}
	out, err := BuildConfig(context.Background(), opt, &ReadOptions{Options: &input})
	if err != nil {
		t.Fatal(err)
	}
	var selector *option.SelectorOutboundOptions
	for _, ob := range out.Outbounds {
		if ob.Tag == OutboundSelectTag {
			selector = ob.Options.(*option.SelectorOutboundOptions)
		}
	}
	if selector == nil {
		t.Fatal("selector missing")
	}
	if selector.Default != OutboundURLTestTag {
		t.Fatalf("selector default = %q, want %q", selector.Default, OutboundURLTestTag)
	}
	var hasLowest bool
	for _, tag := range selector.Outbounds {
		switch tag {
		case OutboundURLTestTag:
			hasLowest = true
		case OutboundRoundRobinTag:
			t.Fatalf("round-robin balancer must not be selectable: %v", selector.Outbounds)
		}
	}
	if !hasLowest {
		t.Fatalf("lowest-delay balancer must stay selectable: %v", selector.Outbounds)
	}
	for _, ob := range out.Outbounds {
		if ob.Tag == OutboundRoundRobinTag {
			t.Fatalf("round-robin balancer outbound must not be built")
		}
	}
}
