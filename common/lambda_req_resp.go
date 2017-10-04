package common

type UserCertReqJson struct {
	EventType string
	Token Token
	SshUsername string
	PublicKey string
}

type HostCertReqJson struct {
	EventType string
	Token Token
	PublicKey string
}

type UserCertRespJson struct {
	SignedPublicKey string
	Jumpbox *Jumpbox `json:",omitempty"`
	Expiry int64
}

type Jumpbox struct {
	IpAddress  string
	User       string
}

type HostCertRespJson struct {
	SignedHostPublicKey string
}
