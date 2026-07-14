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
	viper.Reset()
	defer viper.Reset()

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
	viper.Set("gateway_pure_mode", true)
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

func TestSyncFromFileDisablesWalletsMissingFromConfig(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	if _, err := data.AddWalletAddressWithNetwork("tron", "TOldWalletShouldDisable001"); err != nil {
		t.Fatalf("seed old wallet: %v", err)
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gateway.yaml")
	content := `
wallets:
  - network: tron
    address: TNewWalletOnly001
    remark: new wallet
    enabled: true
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write gateway config: %v", err)
	}

	viper.Set("gateway_config", configPath)
	viper.Set("gateway_pure_mode", true)
	config.SettingsGetString = func(key string) string {
		return data.GetSettingString(key, "")
	}

	if err := SyncFromFile(); err != nil {
		t.Fatalf("SyncFromFile(): %v", err)
	}

	oldRow, err := data.GetWalletAddressByNetworkAndAddress("tron", "TOldWalletShouldDisable001")
	if err != nil {
		t.Fatalf("reload old wallet: %v", err)
	}
	if oldRow.Status != mdb.TokenStatusDisable {
		t.Fatalf("old wallet status = %d, want %d", oldRow.Status, mdb.TokenStatusDisable)
	}

	rows, err := data.GetAvailableWalletAddressByNetwork("tron")
	if err != nil {
		t.Fatalf("GetAvailableWalletAddressByNetwork(): %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("enabled wallet count = %d, want 1", len(rows))
	}
	if rows[0].Address != "TNewWalletOnly001" {
		t.Fatalf("enabled wallet address = %q, want %q", rows[0].Address, "TNewWalletOnly001")
	}
}

func TestSyncFromFileSkipsWhenPureGatewayModeDisabled(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	cleanup := testutil.SetupTestDatabases(t)
	defer cleanup()

	oldWallet, err := data.AddWalletAddressWithNetwork("tron", "TAdminModeWalletShouldStay001")
	if err != nil {
		t.Fatalf("seed old wallet: %v", err)
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "gateway.yaml")
	content := `
wallets:
  - network: tron
    address: TConfigWalletIgnored001
    enabled: true
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write gateway config: %v", err)
	}

	viper.Set("gateway_config", configPath)
	viper.Set("gateway_pure_mode", false)

	if err := SyncFromFile(); err != nil {
		t.Fatalf("SyncFromFile(): %v", err)
	}

	keptWallet, err := data.GetWalletAddressByNetworkAndAddress("tron", oldWallet.Address)
	if err != nil {
		t.Fatalf("reload old wallet: %v", err)
	}
	if keptWallet.Status != mdb.TokenStatusEnable {
		t.Fatalf("old wallet status = %d, want %d", keptWallet.Status, mdb.TokenStatusEnable)
	}

	ignoredWallet, err := data.GetWalletAddressByNetworkAndAddress("tron", "TConfigWalletIgnored001")
	if err != nil {
		t.Fatalf("reload ignored wallet: %v", err)
	}
	if ignoredWallet.ID != 0 {
		t.Fatalf("config wallet was created in admin mode, id=%d", ignoredWallet.ID)
	}
}
