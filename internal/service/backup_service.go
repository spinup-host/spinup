package service

import (
	"archive/tar"
	"context"
	"embed"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	"github.com/spinup-host/spinup/config"
	"github.com/spinup-host/spinup/internal/dockerservice"
	"github.com/spinup-host/spinup/internal/metastore"
	"github.com/spinup-host/spinup/internal/postgres"
	"github.com/spinup-host/spinup/misc"
	"github.com/spinup-host/spinup/utils"
	"go.uber.org/zap"
	"io"
	"log"
	"os"
	"strconv"
)

// Ideally I would like to keep the modify-pghba.sh script to scripts directory.
// However, Go doesn't support relative directory yet https://github.com/golang/go/issues/46056 !!

//go:embed scripts/modify-pghba.sh
var f embed.FS

const (
	tarPath               = "modify-pghba.tar"
	PREFIXBACKUPCONTAINER = "spinup-postgres-backup-"
)

type BackupService struct {
	store        metastore.Db
	logger       *zap.Logger
	dockerClient dockerservice.Docker
}

func NewBackupService(store metastore.Db, client dockerservice.Docker, logger *zap.Logger) BackupService {
	return BackupService{
		store:        store,
		logger:       logger,
		dockerClient: client,
	}
}

type BackupData struct {
	AwsAccessKeySecret string
	AwsAccessKeyId     string
	WalgS3Prefix       string
	PgHost             string
	PgUsername         string
	PgPassword         string
	PgDatabase         string
}

func (bs BackupService) CreateBackup(ctx context.Context, clusterID string, backupConfig metastore.BackupConfig) error {
	cluster, err := metastore.GetClusterByID(bs.store, clusterID)
	if err != nil {
		return err
	}

	pgHost := postgres.PREFIXPGCONTAINER + cluster.Name
	var pgContainer *dockerservice.Container
	if pgContainer, err = bs.dockerClient.GetContainer(context.Background(), postgres.PREFIXPGCONTAINER+cluster.Name); err != nil {
		return errors.Wrap(err, "failed to get cluster container")
	}

	if pgContainer == nil {
		return errors.New("no container matched the provided ID")
	}

	minute, _ := backupConfig.Schedule["minute"].(string)
	min, _ := strconv.Atoi(minute)

	hour, _ := backupConfig.Schedule["hour"].(string)
	h, _ := strconv.Atoi(hour)

	dom, _ := backupConfig.Schedule["dom"].(string)
	domInt, _ := strconv.Atoi(dom)

	month, _ := backupConfig.Schedule["month"].(string)
	mon, _ := strconv.Atoi(month)

	dow, _ := backupConfig.Schedule["dow"].(string)
	dowInt, _ := strconv.Atoi(dow)

	insertSql := "insert into backup(clusterId, destination, bucket, second, minute, hour, dom, month, dow) values(?, ?, ?, ?, ?, ?, ?, ?, ?)"
	if err := metastore.InsertBackup(
		bs.store,
		insertSql,
		clusterID,
		backupConfig.Dest.Name,
		backupConfig.Dest.BucketName,
		0,
		min,
		h,
		domInt,
		mon,
		dowInt,
	); err != nil {
		return err
	}

	scriptContent, err := f.ReadFile("modify-pghba.sh")
	if err != nil {
		utils.Logger.Error("reading modify-pghba.sh file ", zap.Error(err))
	}
	if err = updatePghba(pgContainer, bs.dockerClient, scriptContent); err != nil {
		return errors.Wrap(err, "failed to update pghba")
	}

	execPath := "/usr/lib/postgresql/" + strconv.Itoa(cluster.MajVersion) + "/bin/"
	if err = postgres.ReloadPostgres(bs.dockerClient, execPath, postgres.PGDATADIR, pgHost); err != nil {
		return errors.Wrap(err, "failed to relaod postgres")
	}
	scheduler := cron.New()
	spec := scheduleToCronExpr(backupConfig.Schedule)
	utils.Logger.Info("Scheduling backup at ", zap.String("spec", spec))

	backupData := BackupData{
		AwsAccessKeySecret: backupConfig.Dest.ApiKeySecret,
		AwsAccessKeyId:     backupConfig.Dest.ApiKeyID,
		WalgS3Prefix:       backupConfig.Dest.BucketName,
		PgHost:             pgHost,
		PgDatabase:         "postgres",
		PgUsername:         cluster.Username,
		PgPassword:         cluster.Password,
	}
	_, err = scheduler.AddFunc(spec, TriggerBackup(config.DefaultNetworkName, backupData))
	if err != nil {
		utils.Logger.Error("scheduling database backup", zap.Error(err))
		return err
	}
	scheduler.Start()
	return nil
}

func scheduleToCronExpr(schedule map[string]interface{}) string {
	spec := ""
	if minute, ok := schedule["minute"].(string); ok {
		spec = minute
	} else {
		spec += " " + "*"
	}
	if hour, ok := schedule["hour"].(string); ok {
		spec += " " + hour
	} else {
		spec += " " + "*"
	}
	if dom, ok := schedule["dom"].(string); ok {
		spec += " " + dom
	} else {
		spec += " " + "*"
	}
	if month, ok := schedule["month"].(string); ok {
		spec += " " + month
	} else {
		spec += " " + "*"
	}
	if dow, ok := schedule["dow"].(string); ok {
		spec += " " + dow
	} else {
		spec += " " + "*"
	}

	return spec
}

func updatePghba(c *dockerservice.Container, d dockerservice.Docker, content []byte) error {
	_, cleanup, err := contentToTar(content)
	if err != nil {
		return errors.Wrap(err, "failed to convert content to tar file")
	}
	defer cleanup()
	tr, err := os.Open(tarPath)
	if err != nil {
		return errors.Wrap(err, "error reading tar file")
	}
	defer tr.Close()
	err = d.Cli.CopyToContainer(context.Background(), c.ID, "/etc/postgresql", tr, types.CopyToContainerOptions{})
	if err != nil {
		return errors.Wrap(err, "error copying file to container")
	}
	hbaFile := postgres.PGDATADIR + "pg_hba.conf"
	execConfig := types.ExecConfig{
		User:         "postgres",
		WorkingDir:   "/etc/postgresql",
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          []string{"./modify-pghba", hbaFile},
	}
	if _, err := c.ExecCommand(context.Background(), d, execConfig); err != nil {
		return errors.Wrapf(err, "error executing command '%s'", execConfig.Cmd[0])
	}
	return nil
}

// contentToTar returns a tar file for given content
// ref https://medium.com/learning-the-go-programming-language/working-with-compressed-tar-files-in-go-e6fe9ce4f51d
func contentToTar(content []byte) (io.Writer, func(), error) {
	tarFile, err := os.Create(tarPath)
	if err != nil {
		log.Fatal(err)
	}
	defer tarFile.Close()
	tw := tar.NewWriter(tarFile)
	defer tw.Close()

	hdr := &tar.Header{
		Name: "modify-pghba",
		Mode: 0655,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, nil, err
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		return nil, nil, err
	}
	rmFunc := func() {
		os.Remove(tarPath)
	}
	return tw, rmFunc, nil
}

// TriggerBackup returns a closure which is being invoked by the cron
func TriggerBackup(networkName string, backupData BackupData) func() {
	var err error
	dockerClient, err := dockerservice.NewDocker(config.DefaultNetworkName)
	if err != nil {
		utils.Logger.Error("Error creating client", zap.Error(err))
	}
	var op container.ContainerCreateCreatedBody
	env := []string{
		misc.StringToDockerEnvVal("AWS_SECRET_ACCESS_KEY", backupData.AwsAccessKeySecret),
		misc.StringToDockerEnvVal("AWS_ACCESS_KEY_ID", backupData.AwsAccessKeyId),
		misc.StringToDockerEnvVal("WALG_S3_PREFIX", backupData.WalgS3Prefix),
		misc.StringToDockerEnvVal("PGHOST", backupData.PgHost),
		misc.StringToDockerEnvVal("PGPASSWORD", backupData.PgPassword),
		misc.StringToDockerEnvVal("PGDATABASE", backupData.PgDatabase),
		misc.StringToDockerEnvVal("PGUSER", backupData.PgUsername),
	}

	// Ref: https://gist.github.com/viggy28/5b524baf005d029e4bad2ec16cb09dca
	// On dealing with container networking and environment variables
	// initialized a map
	endpointConfig := map[string]*network.EndpointSettings{}
	// setting key and value for the map. networkName=$dbname_default (eg: viggy_default)
	endpointConfig[networkName] = &network.EndpointSettings{}
	nwConfig := network.NetworkingConfig{EndpointsConfig: endpointConfig}
	backupContainer := dockerservice.NewContainer(
		PREFIXBACKUPCONTAINER+backupData.PgHost,
		container.Config{
			Image:        "spinuphost/walg:latest",
			Env:          env,
			ExposedPorts: map[nat.Port]struct{}{"5432": {}},
		},
		container.HostConfig{NetworkMode: "default"},
		nwConfig,
	)
	return func() {
		utils.Logger.Info("starting backup")
		op, err = backupContainer.Start(context.Background(), dockerClient)
		if err != nil {
			utils.Logger.Error("starting backup container", zap.Error(err))
		}
		utils.Logger.Info("created backup container:", zap.String("containerId", op.ID))
		utils.Logger.Info("Ending backup")
	}
}
