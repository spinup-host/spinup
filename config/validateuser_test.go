package config

import (
	"testing"
)

func TestValidateUser(t *testing.T) {
	Cfg.Common.ApiKey = "test_api_key"
	t.Run("invalid user", func(t *testing.T) {
		msg, err := ValidateUser("", "")
		validErrMsg := "no authorization keys found"
		if err.Error() != validErrMsg || msg != "" {
			t.Errorf("expected: %s ,found: %s ,userId: %s", validErrMsg, err.Error(), msg)
		}
		invalidApiKey := Cfg.Common.ApiKey + "$"
		msg, err = ValidateUser("", invalidApiKey)
		validErrMsg = "error validating api-key"
		if err.Error() != validErrMsg || msg != "" {
			t.Errorf("expected: %s ,found: %s ,userId: %s", validErrMsg, err.Error(), msg)
		}
	})

	t.Run("valid user", func(t *testing.T) {
		userId, err := ValidateUser("", Cfg.Common.ApiKey)
		if err != nil || userId != "testuser" {
			t.Errorf("expected: testuser ,found: %s ,userId: %s", err.Error(), userId)
		}
	})

}
