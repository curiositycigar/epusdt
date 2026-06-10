package gateway

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GMWalletApp/epusdt/config"
	"github.com/GMWalletApp/epusdt/internal/testutil"
	"github.com/GMWalletApp/epusdt/model/data"
	"github.com/GMWalletApp/epusdt/model/mdb"
	"github.com/spf13/viper"
)

func TestSyncFromFileSeedsMerchantWalletAndSettings(t *testing.T) {
	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gateway.yaml")
	content := `
merchants:
  - name: app-a
    pid: "2000"
    secret_key: "secret-2000"
    notify_url: "https://merchant.example/notify"
    enabled: true
wallets:
  - network: tron
    address: TGatewaySyncWallet001
    remark: synced wallet
    enabled: true
settings:
  - group: rate
    key: rate.forced_rate_list
    value: '{"cny":{"usdt":0.25}}'
    type: json
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write gateway config: %v", err)
	}

	viper.Set("gateway_config", configPath)
	config.SettingsGetString = func(key string) string {
		return data.GetSettingString(key, "")
	}

	if err := SyncFromFile(); err != nil {
		t.Fatalf("SyncFromFile(): %v", err)
	}

	keyRow, err := data.GetEnabledApiKey("2000")
	if err != nil {
		t.Fatalf("GetEnabledApiKey(): %v", err)
	}
	if keyRow.ID == 0 {
		t.Fatal("merchant api key not created")
	}
	if keyRow.SecretKey != "secret-2000" {
		t.Fatalf("secret_key = %q, want %q", keyRow.SecretKey, "secret-2000")
	}

	walletRow, err := data.GetWalletAddressByNetworkAndAddress("tron", "TGatewaySyncWallet001")
	if err != nil {
		t.Fatalf("GetWalletAddressByNetworkAndAddress(): %v", err)
	}
	if walletRow.ID == 0 {
		t.Fatal("wallet not created")
	}
	if walletRow.Status != mdb.TokenStatusEnable {
		t.Fatalf("wallet status = %d, want %d", walletRow.Status, mdb.TokenStatusEnable)
	}

	if got := data.GetSettingString(mdb.SettingKeyRateForcedRateList, ""); got != `{"cny":{"usdt":0.25}}` {
		t.Fatalf("rate.forced_rate_list = %q, want synced value", got)
	}
}
