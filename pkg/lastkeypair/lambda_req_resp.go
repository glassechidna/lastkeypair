package lastkeypair

type UserCertReqJson struct {
	// NOTE: be very careful of adding new fields to this struct. only fields
	// inside Token.TokenParams are part of the encryption context (and hence
	// logged in cloudtrail)
	EventType string
	Token Token
	PublicKey string
}

type HostCertReqJson struct {
	EventType string
	Token Token
	PublicKey string
}

type UserCertRespJson struct {
	SignedPublicKey string
	Jumpboxes []Jumpbox `json:",omitempty"`
	TargetAddress string `json:",omitempty"`
	Expiry int64
}

type Jumpbox struct {
	Address    string
	User       string
	HostKeyAlias string
	Principals []string
	SignedPublicKey string
	CertificateOptions *CertificateOptions
}

type CertificateOptions struct {
	ForceCommand  *string `json:",omitempty"`
	SourceAddress *string `json:",omitempty"`
	PermitX11Forwarding bool
	PermitAgentForwarding bool
	PermitPortForwarding bool
}

type HostCertRespJson struct {
	SignedHostPublicKey string
}
