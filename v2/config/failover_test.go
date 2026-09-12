package config

import (
	"context"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

func TestDefaultFailoverOptions(t *testing.T) {
	o := DefaultHiddifyOptions()
	if o.Failover.Tolerance != 150 || o.Failover.MinDwell != 60 || o.Failover.StallTimeout != 8 || o.Failover.StallThreshold != 3 ||
		o.Failover.StallWindow != 30 || o.Failover.RescueBatch != 6 || o.Failover.RescueTimeout != 5 || o.Failover.ActiveCheckInterval != 180 {
		t.Fatalf("defaults: %+v", o.Failover)
	}
	if !o.DisableInterfaceSweep || o.URLTestInterval != DurationInSeconds(1800) {
		t.Fatalf("sweep defaults: disable=%v interval=%v", o.DisableInterfaceSweep, o.URLTestInterval)
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
