package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type backupOpts struct {
	logger     *zap.Logger
	clusterId  string
	backupName string
}

func backupCmd() *cobra.Command {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Println("failed to build logger: ", err.Error())
	}

	backupOptions := &backupOpts{
		logger: logger,
	}
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
	b.logger.Info("attempting restore", zap.String("backup_name", b.backupName), zap.String("cluster_id", b.clusterId))

}
