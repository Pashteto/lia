package links

import "fmt"

//go:generate ../../../bin/mockery --name LinkBuilder
type LinkBuilder interface {
	GetReferralLink(refCode string) string
}

type linkBuilder struct {
	referralLinkFormat string
}

func NewLinkBuilder(referralLinkFormat string) *linkBuilder {
	return &linkBuilder{referralLinkFormat: referralLinkFormat}
}

func (b *linkBuilder) GetReferralLink(refCode string) string {
	return fmt.Sprintf(b.referralLinkFormat, refCode)
}
