package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/spinup-host/spinup/config"
	"github.com/spinup-host/spinup/internal/metastore"
	"github.com/spinup-host/spinup/utils"
)

// createBackupRequest holds the parameters needed to create a backup
type createBackupRequest struct {
	ClusterID    string `json:"cluster_id"`
	Name         string `json:"name"`
	ApiKeyID     string `json:"api_key_id"`
	ApiKeySecret string `json:"api_key_secret"`
	BucketName   string `json:"bucket_name"`
}

type BackupHandler struct {
	logger        *zap.Logger
	appConfig     config.Configuration
	backupService backupService
}

func NewBackupHandler(cfg config.Configuration, backupService backupService, logger *zap.Logger) BackupHandler {
	return BackupHandler{
		logger:        logger,
		appConfig:     cfg,
		backupService: backupService,
	}
}

func (b BackupHandler) CreateBackup(w http.ResponseWriter, r *http.Request) {
	if (*r).Method != "POST" {
		respond(http.StatusMethodNotAllowed, w, map[string]string{"message": "invalid method"})
		return
	}
	var s createBackupRequest
	byteArray, err := io.ReadAll(r.Body)
	if err != nil {
		utils.Logger.Error("failed to read request body", zap.Error(err))
		respond(http.StatusInternalServerError, w, map[string]string{"message": "failed to read request body"})
		return
	}
	err = json.Unmarshal(byteArray, &s)
	if err != nil {
		utils.Logger.Error("failed to parse request body", zap.Error(err))
		respond(http.StatusInternalServerError, w, map[string]string{"message": "failed to parse request body"})
		return
	}

	if err = backupDataValidation(s); err != nil {
		l := &logicError{}
		if errors.As(err, l) {
			respond(http.StatusBadRequest, w, map[string]string{"message": l.Error()})
		} else {
			respond(http.StatusInternalServerError, w, map[string]string{"message": err.Error()})
		}
		return
	}

	backupCfg := metastore.BackupConfig{
		Dest: metastore.Destination{
			Name:         s.Name,
			BucketName:   s.BucketName,
			ApiKeyID:     s.ApiKeyID,
			ApiKeySecret: s.ApiKeySecret,
		},
	}

	if err := b.backupService.CreateBackup(r.Context(), s.ClusterID, backupCfg); err != nil {
		b.logger.Error("failed to create backup", zap.Error(err))
		respond(http.StatusInternalServerError, w, map[string]string{"message": err.Error()})
		return
	}
	respond(http.StatusOK, w, map[string]string{"message": "successfully scheduled backup"})
}

type logicError struct {
	err error
}

func (l logicError) Error() string {
	return fmt.Sprintf("logic error %v", l.err)
}

func backupDataValidation(s createBackupRequest) error {
	if s.ClusterID == "" {
		return errors.New("no cluster specified as a backup target")
	}

	if s.Name != "AWS" {
		return logicError{err: errors.New("destination other than AWS is not supported")}
	}
	if s.ApiKeyID == "" || s.ApiKeySecret == "" {
		return errors.New("api key id and api key secret is mandatory")
	}
	if s.BucketName == "" {
		return errors.New("bucket name is mandatory")
	}
	return nil
}
