package api

import (
	"testing"

	"github.com/spinup-host/spinup/config"
)

func TestValidateUser(t *testing.T) {
	cfg := config.Configuration{}
	cfg.Common.ApiKey = "test_api_key"
	t.Run("invalid user", func(t *testing.T) {
		msg, err := ValidateUser(cfg, "", "")
		validErrMsg := "no authorization keys found"
		if err.Error() != validErrMsg || msg != "" {
			t.Errorf("expected: %s ,found: %s ,userId: %s", validErrMsg, err.Error(), msg)
		}
		invalidApiKey := cfg.Common.ApiKey + "$"
		msg, err = ValidateUser(cfg, "", invalidApiKey)
		validErrMsg = "error validating api-key"
		if err.Error() != validErrMsg || msg != "" {
			t.Errorf("expected: %s ,found: %s ,userId: %s", validErrMsg, err.Error(), msg)
		}
	})

	t.Run("valid user", func(t *testing.T) {
		userId, err := ValidateUser(cfg, "", cfg.Common.ApiKey)
		if err != nil || userId != "testuser" {
			t.Errorf("expected: testuser ,found: %s ,userId: %s", err.Error(), userId)
		}
	})

}
