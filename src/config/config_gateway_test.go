package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestBuildPaymentURLUsesTemplateWhenConfigured(t *testing.T) {
	viper.Reset()
	viper.Set("gateway_payment_url_template", "https://merchant.example/pay/{trade_id}")

	got := BuildPaymentURL("trade-123")
	if got != "https://merchant.example/pay/trade-123" {
		t.Fatalf("BuildPaymentURL() = %q, want %q", got, "https://merchant.example/pay/trade-123")
	}
}

func TestBuildPaymentURLEmptyWhenUIIsDisabledAndNoTemplate(t *testing.T) {
	viper.Reset()
	viper.Set("gateway_pure_mode", true)

	got := BuildPaymentURL("trade-123")
	if got != "" {
		t.Fatalf("BuildPaymentURL() = %q, want empty", got)
	}
}

func TestBuildPaymentURLFallsBackToLocalCashierWhenUIEnabled(t *testing.T) {
	viper.Reset()
	viper.Set("app_uri", "https://pay.example.com")

	got := BuildPaymentURL("trade-123")
	if got != "https://pay.example.com/pay/checkout-counter/trade-123" {
		t.Fatalf("BuildPaymentURL() = %q, want local cashier URL", got)
	}
}

func TestIsPureGatewayModeReadsConfigFileBeforeInit(t *testing.T) {
	oldExplicit := explicitConfigPath
	defer func() { explicitConfigPath = oldExplicit }()

	viper.Reset()
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	if err := os.WriteFile(path, []byte("gateway_pure_mode=true\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	SetConfigPath(path)
	if !IsPureGatewayMode() {
		t.Fatal("IsPureGatewayMode() = false, want true from config file")
	}
}
