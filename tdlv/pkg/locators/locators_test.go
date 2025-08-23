package locators

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocator(t *testing.T) {
	cwd := "/Users/foo/temporal-workflow-replay-debugger/example/go/structured-workflow/replay-debug-ide-integrated"
	file := "/Users/foo/temporal-workflow-replay-debugger/example/go/structured-workflow/workflow-code/pkg/workflows/workflow.go"
	result := IsUserCodeFile(file, cwd)
	assert.True(t, result)
}
