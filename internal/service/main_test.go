package service

import (
	"os"
	"testing"

	"github.com/SukGebeya/Backend/internal/testutil"
)

func TestMain(m *testing.M) {
	code := m.Run()
	testutil.StopSharedPostgres()
	testutil.StopSharedRedis()
	os.Exit(code)
}
