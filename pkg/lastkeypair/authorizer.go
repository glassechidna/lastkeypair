package lastkeypair

import (
	"github.com/aws/aws-sdk-go/service/lambda"
	"encoding/json"
	"github.com/pkg/errors"
)

type authorizationLambdaIdentity struct {
	Name    *string `json:",omitempty"`
	Id      string
	Account string
	Type    string
}

type authorizationLambdaVoucher struct {
	Name    *string `json:",omitempty"`
	Id      string
	Account string
	Type    string
	Vouchee string
	Context string
}

type LkpUserCertAuthorizationRequest struct {
	Kind              string
	From              authorizationLambdaIdentity
	RemoteInstanceArn string
	SshUsername       string
	Vouchers          []authorizationLambdaVoucher `json:",omitempty"`
}

type LkpUserCertAuthorizationResponse struct {
	Authorized bool
	Message string
	Principals []string
	Jumpboxes  []Jumpbox `json:",omitempty"`
	TargetAddress string `json:",omitempty"`
	CertificateOptions struct {
		ForceCommand  *string `json:",omitempty"`
		SourceAddress *string `json:",omitempty"`
		PermitPortForwarding bool
		PermitX11Forwarding bool
	}
}

type LkpHostCertAuthorizationRequest struct {
	Kind            string
	From            authorizationLambdaIdentity
	HostInstanceArn string
	Principals      []string
}

type LkpHostCertAuthorizationResponse struct {
	Authorized bool
	KeyId      string
	Principals []string
}

type AuthorizationLambda struct {
	config LambdaConfig
}

func NewAuthorizationLambda(config LambdaConfig) *AuthorizationLambda {
	return &AuthorizationLambda{config: config}
}

func (a *AuthorizationLambda) doLambda(req interface{}, resp interface{}) error {
	client := lambda.New(LambdaAwsSession())

	encoded, err := json.Marshal(&req)
	if err != nil {
		return errors.Wrap(err, "encoding authorisation lambda request")
	}

	input := &lambda.InvokeInput{
		FunctionName: &a.config.AuthorizationLambda,
		Payload:      encoded,
	}

	lambdaResp, err := client.Invoke(input)
	if err != nil {
		return errors.Wrap(err, "executing authorisation lambda")
	}

	err = json.Unmarshal(lambdaResp.Payload, resp)
	if err != nil {
		return errors.Wrap(err, "decoding auth lambda response")
	}

	return nil
}

func tokenParamsToAuthLambdaIdentity(p TokenParams) authorizationLambdaIdentity {
	return authorizationLambdaIdentity{
		Name:    &p.FromName,
		Id:      p.FromId,
		Account: p.FromAccount,
		Type:    p.Type,
	}
}

func (a *AuthorizationLambda) DoUserReq(userReq UserCertReqJson) (*LkpUserCertAuthorizationResponse, error) {
	if len(a.config.AuthorizationLambda) == 0 {
		return &LkpUserCertAuthorizationResponse{
			Authorized: true,
			Principals: []string{userReq.Token.Params.RemoteInstanceArn},
		}, nil
	}

	p := userReq.Token.Params
	req := LkpUserCertAuthorizationRequest{
		Kind:              "LkpUserCertAuthorizationRequest",
		From:              tokenParamsToAuthLambdaIdentity(p),
		RemoteInstanceArn: p.RemoteInstanceArn,
		SshUsername:       userReq.Token.Params.SshUsername,
	}

	for _, v := range p.Vouchers {
		vp := v.Params
		voucher := authorizationLambdaVoucher{
			Name:    &vp.FromName,
			Id:      vp.FromId,
			Account: vp.FromAccount,
			Type:    vp.Type,
			Vouchee: vp.Vouchee,
			Context: vp.Context,
		}
		req.Vouchers = append(req.Vouchers, voucher)
	}

	authResp := LkpUserCertAuthorizationResponse{}
	err := a.doLambda(req, &authResp)
	if err != nil {
		return nil, errors.Wrap(err, "invoking user cert authorisation lambda")
	}

	jumpPrincipals := []string{}
	for idx := range authResp.Jumpboxes {
		j := &authResp.Jumpboxes[idx]
		if len(j.HostKeyAlias) == 0 {
			j.HostKeyAlias = j.Address
		}
		jumpPrincipals = append(jumpPrincipals, j.HostKeyAlias)
	}

	// if the lambda's response is missing the "Principals" key, default to the requested instance
	if authResp.Principals == nil {
		authResp.Principals = append(jumpPrincipals, p.RemoteInstanceArn)
	}

	return &authResp, nil
}

func (a *AuthorizationLambda) DoHostReq(hostReq HostCertReqJson) (*LkpHostCertAuthorizationResponse, error) {
	hostArn := hostReq.Token.Params.HostInstanceArn

	if len(a.config.AuthorizationLambda) == 0 {
		return &LkpHostCertAuthorizationResponse{
			Authorized: true,
			KeyId:      hostArn,
			Principals: []string{hostArn},
		}, nil
	}

	p := hostReq.Token.Params
	req := LkpHostCertAuthorizationRequest{
		Kind:            "LkpHostCertAuthorizationRequest",
		From:            tokenParamsToAuthLambdaIdentity(p),
		HostInstanceArn: hostArn,
		Principals:      p.Principals,
	}

	authResp := LkpHostCertAuthorizationResponse{}
	err := a.doLambda(req, &authResp)
	if err != nil {
		return nil, errors.Wrap(err, "invoking host cert authorisation lambda")
	}

	if len(authResp.KeyId) == 0 {
		authResp.KeyId = hostArn
	}

	return &authResp, nil
}
