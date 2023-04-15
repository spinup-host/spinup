package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

type backupOpts struct {
	clusterId  string
	backupName string
}

func backupCmd() *cobra.Command {
	backupOptions := &backupOpts{}
	bc := &cobra.Command{
		Use:   "backup",
		Short: "Manage spinup backups",
		Run:   backupOptions.handleBackupCmd,
	}

	//bc.AddCommand(listBackupCmd())
	bc.AddCommand(backupOptions.restoreBackupCmd())
	return bc
}

func (b *backupOpts) restoreBackupCmd() *cobra.Command {
	rbc := &cobra.Command{
		Use:   "restore",
		Short: "Restore a spinup backup",
		Run:   b.handleRestoreBackupCmd,
	}

	rbc.Flags().StringVar(&b.clusterId, "cluster-id", "", "ID of cluster to restore backup into.")
	rbc.Flags().StringVar(&b.backupName, "from", "", "Name of the backup to restore from.")

	return rbc
}

func (b *backupOpts) handleBackupCmd(cmd *cobra.Command, args []string) {
	log.Println("N/A")
}

func (b *backupOpts) handleRestoreBackupCmd(cobra *cobra.Command, args []string) {
	log.Println(fmt.Sprintf("attempting restore into %s", b.clusterId))
}
