// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/glassechidna/lastkeypair/common"
	"encoding/json"
	"io/ioutil"
	"log"
	"fmt"
	"os"
)

// lambdaCmd represents the lambda command
var lambdaCmd = &cobra.Command{
	Use:   "lambda",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		inputPath, _ := cmd.PersistentFlags().GetString("input-path")
		input, _ := ioutil.ReadFile(inputPath)
		os.Setenv("CA_KEY_BYTES",
			`-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA1dKx94oWPCgUlRLmNrhscldKs03BMyWoiEMXwPw+U5Z4JBmo
6dYDdqcE7vHvt0QGYHLJCvtGSQkFCk+1d6aUcy8yBLeqUPam8w7bYEqZCx9h4wtY
ncWPtvC1ArHLZwD7fSjzU62xTxP2c+a3OIA55MVM/hs+z+r3Bo5lWHdfSaHARbO1
OqnNBns3/h2OdBjxzQzEPFj9RL/Gh5Tu0asUQcMAxG6UD9ANzebwjbJf1YhHiZVy
n7s0HASBcw6Dp/PCePRexadZTKuAwRFn5HbIm4kJ9/e6A8UjXK5qNIVm7xk6nZSp
dyZUYgEMvQx79bMygLWgUFeN52VjtdsoopQrUQIDAQABAoIBAGBQ7UuVDxj/8O2J
utuxTWBgA80q3Dk+4GCo4D1VInoikHGqgVT7y0maSHWd055Y7QprCjaBI5LoljWj
3BlOlxYfj0diuDyKLn/UFGuWjsPc2goc5UkEYg1E9jSFhBsc7Sve02TBG9qEIoLo
zWWNFQcA/QKFoVClBasVX39vHiQa/pXjPgzFg1akwc2ZF9xHxFuFEzrL+yid8jOS
qFLVypnHRx3b3zUBd/HxWjK1+Zm/W6+HMmQr+pHA+7EqkMC4P9wYHlU5kSS2Z3/o
iq5gQDt3DMXxxcxsKEic/ts3yNTdyiz/s8rnYTfD+2JgXhLAlYdThCalSzEmI2qa
MfU5UPUCgYEA8IlZn+tsruaYcT7oY2B0AgX5UgHOuwYos2sXR1LHSjBdVA2DKetj
Reu0ylU7Ras64m4rNGziJJM8pfgmetkv/q8hz90iIfOSq3DUet2J8bfd2eJr/TaI
fCtVB8GckbrB7ijeu1Kr8VtpFsKrQ2pi4iszd3bmKgxiIUNgBT+7PrMCgYEA45G0
m88VBLKPSgR2JpTdq68qAPM9H6FTheDCwISn0fIX0fndrYyX44IiyBrklu3Bl900
X941PnxXLk2YEo+bpxleeVI6+4KfGMb7AahwGVxwNXpiqVTzbB9ch0I/G6pyCI6+
k+yw4RqLsGAIXaQeNuRKA6VfpiOMpooN50l2b+sCgYEA5L2JCIZGZZEOuOrc7dxE
lcP+k9j6Mmqp++1ERuRWdpvFtO/gotWhI1YCKEOjSR6LsdaYqZM9/xAxpZd1aG/v
r1/2ZIjjM6xA914mAe15h++VPuWOUk8wvfwrMWQSM5eJYqVlInh84NpP9oALg+HA
xVnV6K6eNLBwBTfgMT2pH/cCgYEA2FfMu9NCyBR45IUZTdR4aJ8972lO0qMsJDpo
610xrgXZX2WLuVHPlBpDtrjaWCHvydAh2oIFXEIZH4vk5sBf2ZvklLH4IOMtHQEN
36Wh7HpUsoKHCTQZttCZxnzUQhjoD/qkczyxa08xPZwOV/eOQeEF/DFbnTZuoGTe
kuLkFcECgYADm0B3k/dpjWqe9y2shDmVcm1My73PZe6Nxm3sFDj5qiwEmClfRjSq
j7mxX8I9N0xyyudbKUcqVCWA1/kYu4LFAZ1PfTx9Oq5Q/wcUhEvLmQKRUMsHrWcw
QdJ7qcO9p+dXCGM3blvNshAIghPyxfDADX4bcDoUG0w60YuF9rpK1A==
-----END RSA PRIVATE KEY-----`)

		doit(input)
	},
}

func doit(input []byte) {
	evt := json.RawMessage(input)
	output, err := common.LambdaHandle(evt, nil)
	if err != nil {
		log.Panicf("err: %+v\n", err)
	}
	fmt.Printf("%+v\n", output)
}

func init() {
	RootCmd.AddCommand(lambdaCmd)

	lambdaCmd.PersistentFlags().String("input-path", "", "")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// lambdaCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// lambdaCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
