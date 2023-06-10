package cmd

import (
	"log"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/spinup-host/spinup/config"
	"github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/internal/metastore"
	"github.com/spinup-host/spinup/internal/service"
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
	if err := validateConfig(cfgFile); err != nil {
		b.logger.Fatal("failed to validate config: %v", zap.Error(err))
		return
	}

	b.logger.Info("attempting restore", zap.String("backup_name", b.backupName), zap.String("cluster_id", b.clusterId))
	dockerClient, err := dockerservice.NewDocker(config.DefaultNetworkName)
	if err != nil {
		b.logger.Error("could not create docker client", zap.Error(err))
		return
	}
	projectDir := filepath.Join(appConfig.Common.ProjectDir, "metastore.db")
	db, err := metastore.NewDb(projectDir)
	if err != nil {
		b.logger.Fatal("unable to setup sqlite database", zap.Error(err))
		return
	}

	backupService := service.NewBackupService(db, dockerClient, b.logger.Named("backup-service"))
	err = backupService.Restore(cobra.Context(), dockerClient.NetworkName, b.clusterId, b.backupName)
	if err != nil {
		b.logger.Error("failed to complete restore", zap.Error(err))
		return
	}
	b.logger.Info("backup was successfully restored")
}
