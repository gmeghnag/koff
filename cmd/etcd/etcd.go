package etcd

import (
	"os"

	"github.com/spf13/cobra"
	"go.etcd.io/etcd/etcdutl/v3/etcdutl"
)

var OutputFormat string

var EtcdCmd = &cobra.Command{
	Use: "etcd",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
		os.Exit(1)
	},
}

func init() {
	EtcdCmd.AddCommand(
		SnapshotStatus,
		Inspect,
	)
}

var SnapshotStatus = &cobra.Command{
	Use:    "snapshot-status",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		etcdutl.SnapshotStatusCommandFunc(cmd, args)
	},
}

func init() {
	SnapshotStatus.PersistentFlags().StringVarP(&OutputFormat, "write-out", "w", "json", "set the output format (fields, json, simple, table)")

}
