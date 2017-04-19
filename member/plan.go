package member

import (
	_ "github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/plan"
	"regexp"
)

var plan_rexp = regexp.MustCompile(`^([\w]+)-([\w]+)-([0-9]+)-([0-9]*)(day|week|month|year)$`)

func Plan_category(plan_id string) string {
	if m := plan_rexp.FindStringSubmatch(plan_id); m != nil {
		return m[1]
	}
	return ""
}

func Plan_identifier(plan_id string) string {
	if m := plan_rexp.FindStringSubmatch(plan_id); m != nil {
		return m[2]
	}
	return ""
}

func (ms *Members) load_plans() {
	i := plan.List(nil)
	for i.Next() {
		p := i.Plan()
		ms.Plans[Plan_category(p.ID) + "-" + Plan_identifier(p.ID)] = p
	}
}
