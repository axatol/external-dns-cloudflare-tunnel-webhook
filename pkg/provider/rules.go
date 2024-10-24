package provider

import (
	"fmt"

	"github.com/cloudflare/cloudflare-go"
	"github.com/rs/zerolog/log"
	"sigs.k8s.io/external-dns/plan"
)

type Rules []cloudflare.UnvalidatedIngressRule

func (r *Rules) CreateRule(hostname, service string) error {
	for _, rule := range *r {
		if rule.Hostname == hostname && rule.Service == service {
			log.Debug().Str("hostname", hostname).Str("service", service).Msg("rule already exists, skipping")
			return nil
		}

		if rule.Hostname == hostname && rule.Service != service {
			return fmt.Errorf("rule for hostname %s already exists: %s", hostname, service)
		}
	}

	rule := cloudflare.UnvalidatedIngressRule{
		Hostname: hostname,
		Service:  service,
	}

	*r = append([]cloudflare.UnvalidatedIngressRule{rule}, *r...)

	return nil
}

func (r *Rules) UpdateRule(hostname, service string) error {
	for i, rule := range *r {
		if rule.Hostname == hostname {
			(*r)[i].Service = service
			return nil
		}
	}

	return fmt.Errorf("rule for hostname %s does not exist", hostname)
}

func (r *Rules) DeleteRule(hostname string) error {
	for i, rule := range *r {
		if rule.Hostname == hostname {
			*r = append((*r)[:i], (*r)[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("rule for hostname %s does not exist", hostname)
}

func (r *Rules) ApplyChanges(changes *plan.Changes) error {
	for _, change := range changes.Create {
		if err := r.CreateRule(change.DNSName, change.Targets[0]); err != nil {
			return err
		}
	}

	for _, change := range changes.UpdateNew {
		if err := r.UpdateRule(change.DNSName, change.Targets[0]); err != nil {
			return err
		}
	}

	for _, change := range changes.Delete {
		if err := r.DeleteRule(change.DNSName); err != nil {
			return err
		}
	}

	return nil
}
