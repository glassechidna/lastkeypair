package cmd

import (
	"os"
	"net"
	"github.com/spf13/cobra"
	"github.com/glassechidna/lastkeypair/common"
	"github.com/glassechidna/lastkeypair/common/netcat"
)

var sshProxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Internal command invoked by SSH client",
	Long: `
This is used by ssh as the SSH ProxyCommand in order to connect to EC2 instances
by their instance ARN rather than IP address.
`,
	Run: proxy,
}

func proxy(cmd *cobra.Command, args []string) {
	port, _ := cmd.PersistentFlags().GetString("port")

	rei := common.NewReifiedLoginWithCmd(cmd, args)
	rei.PopulateByRestoreCache()

	conn, _ := net.Dial("tcp", rei.Response.TargetAddress + ":" + port)
	netcat.TcpToPipes(conn, os.Stdin, os.Stdout)
}

func init() {
	sshCmd.AddCommand(sshProxyCmd)
	sshProxyCmd.PersistentFlags().String("instance-arn", "", "Fully-specified EC2 instance ARN")
	sshProxyCmd.PersistentFlags().String("port", "22", "Remote SSH server port (default 22)")
}

