package common

import (
	"github.com/aws/aws-sdk-go/service/lambda"
	"encoding/json"
	"github.com/pkg/errors"
)

type AuthorizationLambdaIdentity struct {
	Name    *string `json:",omitempty"`
	Id      string
	Account string
	Type    string
}

type AuthorizationLambdaVoucher struct {
	Name    *string `json:",omitempty"`
	Id      string
	Account string
	Type    string
	Vouchee	string
	Context string
}

type AuthorizationLambdaRequest struct {
	From AuthorizationLambdaIdentity
	RemoteInstanceArn string
	SshUsername string
	Vouchers []AuthorizationLambdaVoucher `json:",omitempty"`
}

type AuthorizationLambdaResponse struct {
	Authorized bool
	Principals []string
	Jumpboxes []Jumpbox `json:",omitempty"`
	CertificateOptions struct {
		ForceCommand *string `json:",omitempty"`
		SourceAddress *string `json:",omitempty"`
	}
}

func DoAuthorizationLambda(userReq UserCertReqJson, config LambdaConfig) (*AuthorizationLambdaResponse, error) {
	if len(config.AuthorizationLambda) == 0 {
		return &AuthorizationLambdaResponse{
			Authorized: true,
			Principals: []string{userReq.Token.Params.RemoteInstanceArn},
		}, nil
	}

	client := lambda.New(LambdaAwsSession())

	p := userReq.Token.Params
	req := AuthorizationLambdaRequest{
		From: AuthorizationLambdaIdentity{
			Name: &p.FromName,
			Id: p.FromId,
			Account: p.FromAccount,
			Type: p.Type,
		},
		RemoteInstanceArn: p.RemoteInstanceArn,
		SshUsername: userReq.Token.Params.SshUsername,
	}

	for _, v := range p.Vouchers {
		vp := v.Params
		voucher := AuthorizationLambdaVoucher{
			Name: &vp.FromName,
			Id: vp.FromId,
			Account: vp.FromAccount,
			Type: vp.Type,
			Vouchee: vp.Vouchee,
			Context: vp.Context,
		}
		req.Vouchers = append(req.Vouchers, voucher)
	}

	encoded, err := json.Marshal(&req)
	if err != nil {
		return nil, errors.Wrap(err, "encoding authorisation lambda request")
	}

	input := &lambda.InvokeInput{
		FunctionName: &config.AuthorizationLambda,
		Payload: encoded,
	}

	resp, err := client.Invoke(input)
	if err != nil {
		return nil, errors.Wrap(err, "executing authorisation lambda")
	}

	authResp := AuthorizationLambdaResponse{}
	err = json.Unmarshal(resp.Payload, &authResp)
	if err != nil {
		return nil, errors.Wrap(err, "decoding auth lambda response")
	}

	return &authResp, nil
}
