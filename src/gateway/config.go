package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/GMWalletApp/epusdt/config"
	"github.com/GMWalletApp/epusdt/model/dao"
	"github.com/GMWalletApp/epusdt/model/data"
	"github.com/GMWalletApp/epusdt/model/mdb"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

type FileConfig struct {
	Merchants            []MerchantConfig            `yaml:"merchants" json:"merchants"`
	Wallets              []WalletConfig              `yaml:"wallets" json:"wallets"`
	Chains               []ChainConfig               `yaml:"chains" json:"chains"`
	ChainTokens          []ChainTokenConfig          `yaml:"chain_tokens" json:"chain_tokens"`
	RPCNodes             []RPCNodeConfig             `yaml:"rpc_nodes" json:"rpc_nodes"`
	Settings             []SettingConfig             `yaml:"settings" json:"settings"`
	NotificationChannels []NotificationChannelConfig `yaml:"notification_channels" json:"notification_channels"`
}

type MerchantConfig struct {
	Name        string   `yaml:"name" json:"name"`
	Pid         string   `yaml:"pid" json:"pid"`
	SecretKey   string   `yaml:"secret_key" json:"secret_key"`
	NotifyURL   string   `yaml:"notify_url" json:"notify_url"`
	IPWhitelist []string `yaml:"ip_whitelist" json:"ip_whitelist"`
	Enabled     *bool    `yaml:"enabled" json:"enabled"`
}

type WalletConfig struct {
	Network string `yaml:"network" json:"network"`
	Address string `yaml:"address" json:"address"`
	Remark  string `yaml:"remark" json:"remark"`
	Source  string `yaml:"source" json:"source"`
	Enabled *bool  `yaml:"enabled" json:"enabled"`
}

type ChainConfig struct {
	Network          string `yaml:"network" json:"network"`
	DisplayName      string `yaml:"display_name" json:"display_name"`
	Enabled          *bool  `yaml:"enabled" json:"enabled"`
	MinConfirmations *int   `yaml:"min_confirmations" json:"min_confirmations"`
	ScanIntervalSec  *int   `yaml:"scan_interval_sec" json:"scan_interval_sec"`
	Extra            string `yaml:"extra" json:"extra"`
}

type ChainTokenConfig struct {
	Network         string   `yaml:"network" json:"network"`
	Symbol          string   `yaml:"symbol" json:"symbol"`
	ContractAddress string   `yaml:"contract_address" json:"contract_address"`
	Decimals        *int     `yaml:"decimals" json:"decimals"`
	Enabled         *bool    `yaml:"enabled" json:"enabled"`
	MinAmount       *float64 `yaml:"min_amount" json:"min_amount"`
}

type RPCNodeConfig struct {
	Network string `yaml:"network" json:"network"`
	URL     string `yaml:"url" json:"url"`
	Type    string `yaml:"type" json:"type"`
	Weight  *int   `yaml:"weight" json:"weight"`
	APIKey  string `yaml:"api_key" json:"api_key"`
	Enabled *bool  `yaml:"enabled" json:"enabled"`
	Purpose string `yaml:"purpose" json:"purpose"`
	Status  string `yaml:"status" json:"status"`
}

type SettingConfig struct {
	Group string `yaml:"group" json:"group"`
	Key   string `yaml:"key" json:"key"`
	Value string `yaml:"value" json:"value"`
	Type  string `yaml:"type" json:"type"`
}

type NotificationChannelConfig struct {
	Type    string          `yaml:"type" json:"type"`
	Name    string          `yaml:"name" json:"name"`
	Config  json.RawMessage `yaml:"config" json:"config"`
	Events  map[string]bool `yaml:"events" json:"events"`
	Enabled *bool           `yaml:"enabled" json:"enabled"`
}

func SyncFromFile() error {
	path := config.GetGatewayConfigPath()
	if path == "" {
		return nil
	}

	cfg, err := loadFileConfig(path)
	if err != nil {
		return err
	}
	return syncConfig(cfg)
}

func loadFileConfig(path string) (*FileConfig, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	cfg := new(FileConfig)
	if err = yaml.Unmarshal(content, cfg); err != nil {
		return nil, fmt.Errorf("parse gateway config %s: %w", path, err)
	}
	return cfg, nil
}

func syncConfig(cfg *FileConfig) error {
	if cfg == nil {
		return nil
	}
	if err := syncMerchants(cfg.Merchants); err != nil {
		return err
	}
	if err := syncWallets(cfg.Wallets); err != nil {
		return err
	}
	if err := syncChains(cfg.Chains); err != nil {
		return err
	}
	if err := syncChainTokens(cfg.ChainTokens); err != nil {
		return err
	}
	if err := syncRPCNodes(cfg.RPCNodes); err != nil {
		return err
	}
	if err := syncSettings(cfg.Settings); err != nil {
		return err
	}
	if err := syncNotificationChannels(cfg.NotificationChannels); err != nil {
		return err
	}
	return data.ReloadSettings()
}

func syncMerchants(rows []MerchantConfig) error {
	for _, item := range rows {
		pid := strings.TrimSpace(item.Pid)
		if pid == "" {
			continue
		}
		enabled := true
		if item.Enabled != nil {
			enabled = *item.Enabled
		}
		status := mdb.ApiKeyStatusEnable
		if !enabled {
			status = mdb.ApiKeyStatusDisable
		}

		existing, err := data.GetApiKeyByPIDAnyStatus(pid)
		if err != nil {
			return err
		}
		fields := map[string]interface{}{
			"name":         strings.TrimSpace(item.Name),
			"secret_key":   strings.TrimSpace(item.SecretKey),
			"notify_url":   strings.TrimSpace(item.NotifyURL),
			"ip_whitelist": strings.Join(normalizeStringList(item.IPWhitelist), ","),
			"status":       status,
		}
		if existing.ID > 0 {
			if err = data.UpdateApiKeyFields(existing.ID, fields); err != nil {
				return err
			}
			continue
		}

		row := &mdb.ApiKey{
			Name:        strings.TrimSpace(item.Name),
			Pid:         pid,
			SecretKey:   strings.TrimSpace(item.SecretKey),
			NotifyUrl:   strings.TrimSpace(item.NotifyURL),
			IpWhitelist: strings.Join(normalizeStringList(item.IPWhitelist), ","),
			Status:      status,
		}
		if err = data.CreateApiKey(row); err != nil {
			return err
		}
	}
	return nil
}

func syncWallets(rows []WalletConfig) error {
	for _, item := range rows {
		network := strings.ToLower(strings.TrimSpace(item.Network))
		address := strings.TrimSpace(item.Address)
		if network == "" || address == "" {
			continue
		}
		enabled := true
		if item.Enabled != nil {
			enabled = *item.Enabled
		}
		status := int64(mdb.TokenStatusEnable)
		if !enabled {
			status = mdb.TokenStatusDisable
		}

		existing, err := data.GetWalletAddressByNetworkAndAddress(network, address)
		if err != nil {
			return err
		}
		if existing.ID == 0 {
			existing, err = data.AddWalletAddressWithNetwork(network, address)
			if err != nil {
				return err
			}
		}
		fields := map[string]interface{}{
			"remark": strings.TrimSpace(item.Remark),
			"source": normalizeWalletSource(item.Source),
			"status": status,
		}
		if err = dao.Mdb.Model(&mdb.WalletAddress{}).Where("id = ?", existing.ID).Updates(fields).Error; err != nil {
			return err
		}
	}
	return nil
}

func syncChains(rows []ChainConfig) error {
	for _, item := range rows {
		network := strings.ToLower(strings.TrimSpace(item.Network))
		if network == "" {
			continue
		}
		existing, err := data.GetChainByNetwork(network)
		if err != nil {
			return err
		}

		fields := map[string]interface{}{}
		if name := strings.TrimSpace(item.DisplayName); name != "" {
			fields["display_name"] = name
		}
		if item.Enabled != nil {
			fields["enabled"] = *item.Enabled
		}
		if item.MinConfirmations != nil {
			fields["min_confirmations"] = *item.MinConfirmations
		}
		if item.ScanIntervalSec != nil {
			fields["scan_interval_sec"] = *item.ScanIntervalSec
		}
		if extra := strings.TrimSpace(item.Extra); extra != "" {
			fields["extra"] = extra
		}

		if existing.ID > 0 {
			if err = data.UpdateChainFields(network, fields); err != nil {
				return err
			}
			continue
		}

		row := &mdb.Chain{
			Network:          network,
			DisplayName:      fallbackString(strings.TrimSpace(item.DisplayName), network),
			Enabled:          valueOrDefaultBool(item.Enabled, true),
			MinConfirmations: valueOrDefaultInt(item.MinConfirmations, 1),
			ScanIntervalSec:  valueOrDefaultInt(item.ScanIntervalSec, 5),
			Extra:            strings.TrimSpace(item.Extra),
		}
		if err = dao.Mdb.Create(row).Error; err != nil {
			return err
		}
	}
	return nil
}

func syncChainTokens(rows []ChainTokenConfig) error {
	for _, item := range rows {
		network := strings.ToLower(strings.TrimSpace(item.Network))
		symbol := strings.ToUpper(strings.TrimSpace(item.Symbol))
		if network == "" || symbol == "" {
			continue
		}

		existing, err := data.GetChainTokenByNetworkAndSymbol(network, symbol)
		if err != nil {
			return err
		}
		fields := map[string]interface{}{
			"contract_address": strings.TrimSpace(item.ContractAddress),
		}
		if item.Decimals != nil {
			fields["decimals"] = *item.Decimals
		}
		if item.Enabled != nil {
			fields["enabled"] = *item.Enabled
		}
		if item.MinAmount != nil {
			fields["min_amount"] = *item.MinAmount
		}
		if existing.ID > 0 {
			if err = data.UpdateChainTokenFields(existing.ID, fields); err != nil {
				return err
			}
			continue
		}

		row := &mdb.ChainToken{
			Network:         network,
			Symbol:          symbol,
			ContractAddress: strings.TrimSpace(item.ContractAddress),
			Decimals:        valueOrDefaultInt(item.Decimals, 6),
			Enabled:         valueOrDefaultBool(item.Enabled, true),
			MinAmount:       valueOrDefaultFloat(item.MinAmount, 0),
		}
		if err = data.CreateChainToken(row); err != nil {
			return err
		}
	}
	return nil
}

func syncRPCNodes(rows []RPCNodeConfig) error {
	for _, item := range rows {
		network := strings.ToLower(strings.TrimSpace(item.Network))
		nodeURL := strings.TrimSpace(item.URL)
		if network == "" || nodeURL == "" {
			continue
		}

		existing, err := data.GetRPCNodeByNetworkAndURL(network, nodeURL)
		if err != nil {
			return err
		}
		fields := map[string]interface{}{
			"type":    strings.ToLower(strings.TrimSpace(item.Type)),
			"api_key": strings.TrimSpace(item.APIKey),
			"purpose": data.NormalizeRpcNodePurpose(item.Purpose),
		}
		if item.Weight != nil {
			fields["weight"] = *item.Weight
		}
		if item.Enabled != nil {
			fields["enabled"] = *item.Enabled
		}
		if status := strings.TrimSpace(item.Status); status != "" {
			fields["status"] = status
		}
		if existing.ID > 0 {
			if err = data.UpdateRpcNodeFields(existing.ID, fields); err != nil {
				return err
			}
			continue
		}

		row := &mdb.RpcNode{
			Network: network,
			Url:     nodeURL,
			Type:    fallbackString(strings.ToLower(strings.TrimSpace(item.Type)), mdb.RpcNodeTypeHttp),
			Weight:  valueOrDefaultInt(item.Weight, 1),
			ApiKey:  strings.TrimSpace(item.APIKey),
			Enabled: valueOrDefaultBool(item.Enabled, true),
			Purpose: data.NormalizeRpcNodePurpose(item.Purpose),
			Status:  fallbackString(strings.TrimSpace(item.Status), mdb.RpcNodeStatusUnknown),
		}
		if err = data.CreateRpcNode(row); err != nil {
			return err
		}
	}
	return nil
}

func syncSettings(rows []SettingConfig) error {
	for _, item := range rows {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		group := strings.TrimSpace(item.Group)
		if group == "" {
			group = inferSettingGroup(key)
		}
		if err := data.SetSetting(group, key, item.Value, fallbackString(strings.TrimSpace(item.Type), mdb.SettingTypeString)); err != nil {
			return err
		}
	}
	return nil
}

func syncNotificationChannels(rows []NotificationChannelConfig) error {
	for _, item := range rows {
		channelType := strings.ToLower(strings.TrimSpace(item.Type))
		if channelType == "" {
			continue
		}
		configJSON := strings.TrimSpace(string(item.Config))
		if configJSON == "" {
			configJSON = "{}"
		}
		eventsJSON, err := json.Marshal(item.Events)
		if err != nil {
			return err
		}
		enabled := true
		if item.Enabled != nil {
			enabled = *item.Enabled
		}

		existing, err := data.GetNotificationChannelByTypeAndName(channelType, strings.TrimSpace(item.Name))
		if err != nil {
			return err
		}
		fields := map[string]interface{}{
			"config":  configJSON,
			"events":  string(eventsJSON),
			"enabled": enabled,
			"type":    channelType,
			"name":    strings.TrimSpace(item.Name),
		}
		if existing.ID > 0 {
			if err = data.UpdateNotificationChannelFields(existing.ID, fields); err != nil {
				return err
			}
			continue
		}

		row := &mdb.NotificationChannel{
			Type:    channelType,
			Name:    strings.TrimSpace(item.Name),
			Config:  configJSON,
			Events:  string(eventsJSON),
			Enabled: enabled,
		}
		if err = data.CreateNotificationChannel(row); err != nil {
			return err
		}
	}
	return nil
}

func normalizeStringList(items []string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func normalizeWalletSource(source string) string {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case mdb.WalletSourceImport:
		return mdb.WalletSourceImport
	default:
		return mdb.WalletSourceManual
	}
}

func inferSettingGroup(key string) string {
	switch {
	case strings.HasPrefix(key, mdb.SettingGroupBrand+"."):
		return mdb.SettingGroupBrand
	case strings.HasPrefix(key, mdb.SettingGroupRate+"."):
		return mdb.SettingGroupRate
	case strings.HasPrefix(key, mdb.SettingGroupSystem+"."):
		return mdb.SettingGroupSystem
	case strings.HasPrefix(key, mdb.SettingGroupEpay+"."):
		return mdb.SettingGroupEpay
	case strings.HasPrefix(key, mdb.SettingGroupOkPay+"."):
		return mdb.SettingGroupOkPay
	default:
		return mdb.SettingGroupSystem
	}
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func valueOrDefaultBool(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func valueOrDefaultInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func valueOrDefaultFloat(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}

var _ *gorm.DB
