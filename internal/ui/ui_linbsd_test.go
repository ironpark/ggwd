//go:build (freebsd || (linux && !android) || netbsd) && !nintendosdk && !playstation5

package ui

import (
	"testing"

	"github.com/ironpark/ggwd/internal/glfw"
)

func TestSurfaceTargetUnsetWindow(t *testing.T) {
	u := &glfwBackend{window: &glfw.Window{}}
	if _, err := u.surfaceTarget(); err == nil {
		t.Fatal("surfaceTarget() succeeded with uninitialized X11 display/window")
	}
}
