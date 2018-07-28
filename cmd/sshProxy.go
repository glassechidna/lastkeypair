package cmd

import (
	"os"
	"net"
	"github.com/spf13/cobra"
	"github.com/glassechidna/lastkeypair/pkg/lastkeypair"
	"github.com/glassechidna/lastkeypair/pkg/lastkeypair/netcat"
	"syscall"
	"os/exec"
	"fmt"
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

	rei := lastkeypair.NewReifiedLoginWithCmd(cmd, args)
	rei.PopulateByRestoreCache()

	jump := rei.Response.Jumpboxes
	if len(jump) == 0 {
		conn, _ := net.Dial("tcp", rei.Response.TargetAddress + ":" + port)
		netcat.TcpToPipes(conn, os.Stdin, os.Stdout)
	} else {
		sshconfPath := rei.WriteSshConfig()
		lastJumphost := fmt.Sprintf("jump%d", len(jump) - 1)
		sshcmd := []string{"ssh", "-F", sshconfPath, "-W", rei.Response.TargetAddress + ":22", lastJumphost}
		sshPath, _ := exec.LookPath("ssh")
		syscall.Exec(sshPath, sshcmd, os.Environ())
	}
}

func init() {
	sshCmd.AddCommand(sshProxyCmd)
	sshProxyCmd.PersistentFlags().String("instance-arn", "", "Fully-specified EC2 instance ARN")
	sshProxyCmd.PersistentFlags().String("port", "22", "Remote SSH server port (default 22)")
}
