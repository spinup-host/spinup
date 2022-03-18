package backup

import (
	"archive/tar"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
	"github.com/robfig/cron/v3"
	"github.com/spinup-host/config"
	"github.com/spinup-host/internal/dockerservice"
	"github.com/spinup-host/internal/metastore"
	"github.com/spinup-host/internal/postgres"
	"github.com/spinup-host/misc"
)

const PREFIXBACKUPCONTAINER = "spinup-postgres-backup-"

type BackupData struct {
	AwsAccessKeySecret string
	AwsAccessKeyId     string
	WalgS3Prefix       string
	PgHost             string
	PgUsername         string
	PgPassword         string
	PgDatabase         string
}

// Ideally I would like to keep the modify-pghba.sh script to scripts directory.
// However, Go doesn't support relative directory yet https://github.com/golang/go/issues/46056 !!

//go:embed modify-pghba.sh
var f embed.FS

func CreateBackup(w http.ResponseWriter, r *http.Request) {
	if (*r).Method != "POST" {
		http.Error(w, "Invalid Method", http.StatusMethodNotAllowed)
		return
	}
	var s config.Service
	byteArray, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("error %v", err)
		misc.ErrorResponse(w, "error reading from request body", 500)
		return
	}
	err = json.Unmarshal(byteArray, &s)
	if err != nil {
		fmt.Printf("error %v", err)
		misc.ErrorResponse(w, "error reading from readall body", 500)
		return
	}
	if err = backupDataValidation(&s); err != nil {
		l := &logicError{}
		if errors.As(err, l) {
			misc.ErrorResponse(w, l.Error(), http.StatusBadRequest)
		} else {
			misc.ErrorResponse(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	path := config.Cfg.Common.ProjectDir + "/" + s.UserID + "/" + s.UserID + ".db"
	db, err := metastore.NewDb(path)
	if err != nil {
		misc.ErrorResponse(w, "error accessing database", 500)
		return
	}
	minute, _ := s.Backup.Schedule["minute"].(string)
	min, _ := strconv.Atoi(minute)

	hour, _ := s.Backup.Schedule["hour"].(string)
	h, _ := strconv.Atoi(hour)

	dom, _ := s.Backup.Schedule["dom"].(string)
	domInt, _ := strconv.Atoi(dom)

	month, _ := s.Backup.Schedule["month"].(string)
	mon, _ := strconv.Atoi(month)

	dow, _ := s.Backup.Schedule["dow"].(string)
	dowInt, _ := strconv.Atoi(dow)

	insertSql := "insert into backup(clusterId, destination, bucket, second, minute, hour, dom, month, dow) values(?, ?, ?, ?, ?, ?, ?, ?, ?)"
	if err := metastore.InsertBackupIntoMeta(
		db,
		insertSql,
		s.Db.ID,
		s.Backup.Dest.Name,
		s.Backup.Dest.BucketName,
		0,
		min,
		h,
		domInt,
		mon,
		dowInt,
	); err != nil {
		log.Printf("ERROR: executing insert into backup table %v", err)
		misc.ErrorResponse(w, "internal server error", 500)
		return
	}
	pgHost := postgres.PREFIXPGCONTAINER + s.Db.Name
	dockerClient, err := dockerservice.NewDocker()
	if err != nil {
		fmt.Printf("error creating client %v", err)
	}
	pgContainer, err := dockerClient.GetContainer(context.Background(), pgHost)
	if err != nil {
		log.Printf("error: getting container for container name %s %v", pgHost, err)
		misc.ErrorResponse(w, "internal server error", 500)
		return
	}
	scriptContent, err := f.ReadFile("modify-pghba.sh")
	if err != nil {
		log.Printf("error: reading modify-pghba.sh file %v", err)
	}
	err = updatePghba(pgContainer, dockerClient, scriptContent)
	if err != nil {
		log.Printf("error: updating pghba %v", err)
		misc.ErrorResponse(w, "internal server error", 500)
		return
	}
	cis, err := metastore.ClustersInfo(db)
	if err != nil {
		misc.ErrorResponse(w, "internal server error", 500)
		return
	}
	ci, err := cis.FilterByName(s.Db.Name)
	if err != nil {
		misc.ErrorResponse(w, "internal server error", 500)
		return
	}
	execPath := "/usr/lib/postgresql/" + strconv.Itoa(ci.MajVersion) + "/bin/"
	if err = postgres.ReloadPostgres(dockerClient, execPath, postgres.PGDATADIR, pgHost); err != nil {
		misc.ErrorResponse(w, "internal server error", 500)
		return
	}
	c := cron.New()
	var spec string
	if minute, ok := s.Backup.Schedule["minute"].(string); ok {
		spec = minute
	} else {
		spec += " " + "*"
	}
	if hour, ok := s.Backup.Schedule["hour"].(string); ok {
		spec += " " + hour
	} else {
		spec += " " + "*"
	}
	if dom, ok := s.Backup.Schedule["dom"].(string); ok {
		spec += " " + dom
	} else {
		spec += " " + "*"
	}
	if month, ok := s.Backup.Schedule["month"].(string); ok {
		spec += " " + month
	} else {
		spec += " " + "*"
	}
	if dow, ok := s.Backup.Schedule["dow"].(string); ok {
		spec += " " + dow
	} else {
		spec += " " + "*"
	}
	log.Println("INFO: scheduling backup at:", spec)

	networkName := s.Db.Name + "_default"
	backupData := BackupData{
		AwsAccessKeySecret: s.Backup.Dest.ApiKeySecret,
		AwsAccessKeyId:     s.Backup.Dest.ApiKeyID,
		WalgS3Prefix:       s.Backup.Dest.BucketName,
		PgHost:             pgHost,
		PgDatabase:         "postgres",
		PgUsername:         s.Db.Username,
		PgPassword:         s.Db.Password,
	}
	_, err = c.AddFunc(spec, TriggerBackup(networkName, backupData))
	if err != nil {
		log.Printf("error: scheduling database backup")
		misc.ErrorResponse(w, "internal server error", 500)
		return
	}
	c.Start()
	w.WriteHeader(http.StatusOK)
}

type logicError struct {
	err error
}

func (l logicError) Error() string {
	return fmt.Sprintf("logic error %v", l.err)
}

func backupDataValidation(s *config.Service) error {
	if !s.BackupEnabled {
		return logicError{err: errors.New("backup is not enabled")}
	}
	if s.Backup.Dest.Name != "AWS" {
		return logicError{err: errors.New("destination other than AWS is not supported")}
	}
	if s.Backup.Dest.ApiKeyID == "" || s.Backup.Dest.ApiKeySecret == "" {
		return errors.New("api key id and api key secret is mandatory")
	}
	if s.Backup.Dest.BucketName == "" {
		return errors.New("bucket name is mandatory")
	}
	if s.UserID == "" {
		s.UserID = "testuser"
	}
	return nil
}

// TriggerBackup returns a closure which is being invoked by the cron
func TriggerBackup(networkName string, backupData BackupData) func() {
	var err error
	dockerClient, err := dockerservice.NewDocker()
	if err != nil {
		fmt.Printf("error creating client %v", err)
	}
	var op container.ContainerCreateCreatedBody
	env := []string{}

	// TODO: refactor this if possible. Challenge is functions can't grow a slice. ie. can't append inside another function
	env = append(env, misc.StringToDockerEnvVal("AWS_SECRET_ACCESS_KEY", backupData.AwsAccessKeySecret))
	env = append(env, misc.StringToDockerEnvVal("AWS_ACCESS_KEY_ID", backupData.AwsAccessKeyId))
	env = append(env, misc.StringToDockerEnvVal("WALG_S3_PREFIX", backupData.WalgS3Prefix))
	env = append(env, misc.StringToDockerEnvVal("PGHOST", backupData.PgHost))
	env = append(env, misc.StringToDockerEnvVal("PGPASSWORD", backupData.PgPassword))
	env = append(env, misc.StringToDockerEnvVal("PGDATABASE", backupData.PgDatabase))
	env = append(env, misc.StringToDockerEnvVal("PGUSER", backupData.PgUsername))

	// Ref: https://gist.github.com/viggy28/5b524baf005d029e4bad2ec16cb09dca
	// On dealing with container networking and environment variables
	// initialized a map
	endpointConfig := map[string]*network.EndpointSettings{}
	// setting key and value for the map. networkName=$dbname_default (eg: viggy_default)
	endpointConfig[networkName] = &network.EndpointSettings{}
	nwConfig := network.NetworkingConfig{EndpointsConfig: endpointConfig}
	containerOptions := &types.ContainerStartOptions{}
	backupContainer := dockerservice.NewContainer(
		PREFIXBACKUPCONTAINER+backupData.PgHost,
		container.Config{
			Image:        "spinuphost/walg:latest",
			Env:          env,
			ExposedPorts: map[nat.Port]struct{}{"5432": {}},
		},
		container.HostConfig{NetworkMode: "default"},
		nwConfig,
		containerOptions,
	)
	return func() {
		// TODO: explicilty ignoring the output of Start. Since i don't know how to use
		fmt.Println("INFO: starting backup")
		op, err = backupContainer.Start(dockerClient)
		if err != nil {
			fmt.Println("error starting backup container %w", err)
		}
		fmt.Println("INFO: created backup container:", op.ID)
		fmt.Println("INFO: ending backup")
	}
}

var tarPath = "modify-pghba.tar"

func updatePghba(c dockerservice.Container, d dockerservice.Docker, content []byte) error {
	_, cleanup, err := contentToTar(content)
	if err != nil {
		return fmt.Errorf("error converting content to tar file %w", err)
	}
	defer cleanup()
	tr, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("error reading tar file %w", err)
	}
	defer tr.Close()
	err = d.Cli.CopyToContainer(context.Background(), c.ID, "/etc/postgresql", tr, types.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("error copying file to container %w", err)
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
		return fmt.Errorf("error executing command %s %w", execConfig.Cmd[0], err)
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
