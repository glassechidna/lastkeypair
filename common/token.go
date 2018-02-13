package common

import "fmt"

type Token struct {
	Params TokenParams
	Signature []byte
}

type TokenParams struct {
	FromId string
	FromAccount string
	To string
	Type string

	// optional fields below this comment
	FromName string `json:",omitempty"`
	Vouchee string `json:",omitempty"`
	Context string `json:",omitempty"`
	Vouchers []VoucherToken `json:",omitempty"`

	// the reason we have both these fields (rather than overloading one "InstanceArn" field)
	// is because we want to specify a KMS key policy that HostInstanceArn _MUST_ match
	// the ec2:SourceInstanceARN if it exists. if we didn't do this, then anyone _not_ on
	// an instance could request a host cert.
	HostInstanceArn   string `json:",omitempty"` // this field is for when an instance is requesting a host cert
	RemoteInstanceArn string `json:",omitempty"` // this field is for when a user is requesting a user cert for a specific host

	SshUsername string `json:",omitempty"` // username on remote instance that user wants to access
	Principals []string `json:",omitempty"` // additional principals to include in cert
}

func (params *TokenParams) ToKmsContext() map[string]*string {
	// TODO: i think this is a recipe for problems. see issue #24
	iterateParams := func(p *TokenParams, cb func(string, *string)) {
		cb("fromId", &p.FromId)
		cb("fromAccount", &p.FromAccount)
		cb("to", &p.To)
		cb("type", &p.Type)

		if len(p.FromName) > 0 {
			cb("fromName", &p.FromName)
		}

		if len(p.HostInstanceArn) > 0 {
			cb("hostInstanceArn", &p.HostInstanceArn)
		}

		if len(p.RemoteInstanceArn) > 0 {
			cb("remoteInstanceArn", &p.RemoteInstanceArn)
		}

		if len(p.SshUsername) > 0 {
			cb("sshUsername", &p.SshUsername)
		}

		if len(p.Vouchee) > 0 {
			cb("vouchee", &p.Vouchee)
		}

		if len(p.Context) > 0 {
			cb("context", &p.Context)
		}
	}

	context := make(map[string]*string)
	iterateParams(params, func(key string, val *string) {
		context[key] = val
	})

	if len(params.Vouchers) > 0 {
		for i, v := range params.Vouchers {
			keyPrefix := fmt.Sprintf("voucher-%d-", i)

			iterateParams(&v.Params, func(key string, val *string) {
				context[keyPrefix + key] = val
			})
		}
	}

	if len(params.Principals) > 0 {
		for i, principal := range params.Principals {
			principal := principal
			key := fmt.Sprintf("principal-%d", i)
			context[key] = &principal
		}
	}

	return context
}
