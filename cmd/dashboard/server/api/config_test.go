package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TODO test settings update and patch logic
func TestUpdateWholeSettings(t *testing.T) {
	_, err := http.NewRequest("POST", uriDocUpdateSettings, nil)
	require.Nil(t, err)
}
