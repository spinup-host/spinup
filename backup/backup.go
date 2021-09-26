package backup

import (
	"fmt"
	"log"

	"github.com/docker/docker/client"
	"github.com/spinup-host/internal"
)

func TriggerBackup(networkName, awsAccessKey, awsAccessKeyId, pgHost, pgUsername, pgPassword, walgS3Prefix string) func() {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Printf("error creating client %v", err)
	}
	backupSvc := internal.NewPgBackupService(cli, awsAccessKey, awsAccessKeyId, pgHost, walgS3Prefix, networkName, pgUsername, pgPassword)
	return func() {
		fmt.Println("backup triggered")
		err = backupSvc.Start()
		if err != nil {
			log.Printf("error running backup service %v", err)
		}
		fmt.Println("backup finished")
	}
}
