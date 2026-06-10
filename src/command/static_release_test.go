package command

import (
	"testing"

	"github.com/spf13/viper"
)

func TestMaybeReleaseStaticUISkipsInPureGatewayMode(t *testing.T) {
	viper.Reset()
	viper.Set("gateway_pure_mode", true)

	called := false
	old := staticReleaser
	staticReleaser = func() error {
		called = true
		return nil
	}
	defer func() { staticReleaser = old }()

	if err := MaybeReleaseStaticUI(); err != nil {
		t.Fatalf("MaybeReleaseStaticUI(): %v", err)
	}
	if called {
		t.Fatal("static releaser should not run in pure gateway mode")
	}
}
