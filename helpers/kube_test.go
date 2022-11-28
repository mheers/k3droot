package helpers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetRunningPods(t *testing.T) {
	_, err := Init()
	require.Nil(t, err)
	pods, err := K8s.GetRunningPods()
	require.Nil(t, err)
	require.NotEmpty(t, pods)
}
