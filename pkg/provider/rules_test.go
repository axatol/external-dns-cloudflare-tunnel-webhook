package provider_test

import (
	"testing"

	"github.com/axatol/external-dns-cloudflare-tunnel-webhook/pkg/provider"
	"github.com/cloudflare/cloudflare-go"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/external-dns/endpoint"
	"sigs.k8s.io/external-dns/plan"
)

func TestRules_CreateRule(t *testing.T) {
	rules := provider.Rules{
		cloudflare.UnvalidatedIngressRule{
			Hostname: "example.com",
			Service:  "service1",
		},
	}

	// same
	err := rules.CreateRule("example.com", "service1")
	assert.NoError(t, err)
	assert.Len(t, rules, 1)

	// new
	err = rules.CreateRule("example2.com", "service2")
	assert.NoError(t, err)
	assert.Len(t, rules, 2)
}

func TestRules_UpdateRule(t *testing.T) {
	rules := provider.Rules{
		cloudflare.UnvalidatedIngressRule{
			Hostname: "example.com",
			Service:  "service1",
		},
	}

	err := rules.UpdateRule("example.com", "service2")
	assert.NoError(t, err)
	assert.Len(t, rules, 1)

	err = rules.UpdateRule("example2.com", "service3")
	assert.EqualError(t, err, "rule for hostname example2.com does not exist")
	assert.Len(t, rules, 1)
}

func TestRules_DeleteRule(t *testing.T) {
	rules := provider.Rules{
		cloudflare.UnvalidatedIngressRule{
			Hostname: "example.com",
			Service:  "service1",
		},
	}

	err := rules.DeleteRule("example.com")
	assert.NoError(t, err)
	assert.Len(t, rules, 0)

	err = rules.DeleteRule("example2.com")
	assert.EqualError(t, err, "rule for hostname example2.com does not exist")
	assert.Len(t, rules, 0)
}

func TestRules_ApplyChanges(t *testing.T) {
	rules := provider.Rules{{
		Hostname: "example.com",
		Service:  "service1",
	}}

	changes := &plan.Changes{
		Create: []*endpoint.Endpoint{
			{
				DNSName:    "example2.com",
				Targets:    []string{"service2"},
				RecordType: "CNAME",
			},
		},
		UpdateNew: []*endpoint.Endpoint{
			{
				DNSName:    "example.com",
				Targets:    []string{"service3"},
				RecordType: "CNAME",
			},
		},
		Delete: []*endpoint.Endpoint{
			{
				DNSName:    "example2.com",
				Targets:    []string{"service2"},
				RecordType: "CNAME",
			},
		},
	}

	err := rules.ApplyChanges(changes)
	assert.NoError(t, err)
	assert.Equal(t, provider.Rules{{
		Hostname: "example.com",
		Service:  "service3",
	}}, rules)
}
