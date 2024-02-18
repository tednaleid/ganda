package cli

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEchoserver(t *testing.T) {
	results, _ := ParseGandaArgs([]string{"ganda", "echoserver"})
	assert.NotNil(t, results)
	//assert.Equal(t, int64(0), results.context.Retries)

	// TODO start here we want to modify this test to check that we can have a default port
	// and an overridden port this could possibly be collapsed back into the main test
}
