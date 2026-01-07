package config

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

// sharedKeyAuth holds account name and key for SharedKeyCredential
type sharedKeyAuth struct {
	AccountName string
	AccountKey  string
}

// AzureAuthMethod represents the Azure authentication method
type AzureAuthMethod string

const (
	// AzureAuthDefault uses DefaultAzureCredential (managed identity, env vars, CLI)
	AzureAuthDefault AzureAuthMethod = "default"
	// AzureAuthManagedIdentity explicitly uses managed identity
	AzureAuthManagedIdentity AzureAuthMethod = "mi"
	// AzureAuthConnectionString uses a connection string
	AzureAuthConnectionString AzureAuthMethod = "connstr"
	// AzureAuthKey uses a storage account key
	AzureAuthKey AzureAuthMethod = "key"
	// AzureAuthSAS uses a SAS token
	AzureAuthSAS AzureAuthMethod = "sas"
)

// GetAzureCredential creates an Azure credential based on the auth method
func GetAzureCredential(cfg *Config) (interface{}, error) {
	switch AzureAuthMethod(cfg.AzureAuth) {
	case AzureAuthDefault:
		return azidentity.NewDefaultAzureCredential(nil)

	case AzureAuthManagedIdentity:
		return azidentity.NewManagedIdentityCredential(nil)

	case AzureAuthConnectionString:
		// Connection string should be in env var or config
		connStr := os.Getenv("AZURE_STORAGE_CONNECTION_STRING")
		if connStr == "" {
			connStr = cfg.AzureConnectionString
		}
		if connStr == "" {
			return nil, fmt.Errorf("connection string required for connstr auth method")
		}
		return connStr, nil

	case AzureAuthKey:
		// Account key for SharedKeyCredential
		if cfg.AzureAccountKey == "" {
			return nil, fmt.Errorf("account key required for key auth method (use --azure-account-key)")
		}
		if cfg.AzureAccount == "" {
			return nil, fmt.Errorf("account name required for key auth method")
		}
		// Return a special marker so we know to use SharedKeyCredential
		return &sharedKeyAuth{AccountName: cfg.AzureAccount, AccountKey: cfg.AzureAccountKey}, nil

	case AzureAuthSAS:
		// SAS token should be in config
		if cfg.AzureSASToken == "" {
			return nil, fmt.Errorf("SAS token required for sas auth method")
		}
		return cfg.AzureSASToken, nil

	default:
		return nil, fmt.Errorf("unsupported azure auth method: %s", cfg.AzureAuth)
	}
}

// CreateAzureBlobClient creates an Azure Blob Storage client based on configuration
func CreateAzureBlobClient(cfg *Config) (*azblob.Client, error) {
	cred, err := GetAzureCredential(cfg)
	if err != nil {
		return nil, fmt.Errorf("get azure credential: %w", err)
	}

	// Determine account URL
	accountURL := cfg.AzureAccountURL
	if accountURL == "" && cfg.AzureAccount != "" {
		accountURL = fmt.Sprintf("https://%s.blob.core.windows.net/", cfg.AzureAccount)
	}

	// Create client based on credential type
	switch v := cred.(type) {
	case *azidentity.DefaultAzureCredential, *azidentity.ManagedIdentityCredential:
		if accountURL == "" {
			return nil, fmt.Errorf("azure account URL required for credential-based auth")
		}
		return azblob.NewClient(accountURL, v.(azcore.TokenCredential), nil)

	case *sharedKeyAuth:
		// Use SharedKeyCredential for account key
		if accountURL == "" {
			return nil, fmt.Errorf("azure account URL required for key auth")
		}
		sharedKeyCred, err := azblob.NewSharedKeyCredential(v.AccountName, v.AccountKey)
		if err != nil {
			return nil, fmt.Errorf("create shared key credential: %w", err)
		}
		return azblob.NewClientWithSharedKeyCredential(accountURL, sharedKeyCred, nil)

	case string:
		// Connection string or SAS token
		if cfg.AzureAuth == string(AzureAuthConnectionString) {
			return azblob.NewClientFromConnectionString(v, nil)
		}
		// SAS token
		if accountURL == "" {
			return nil, fmt.Errorf("azure account URL required for SAS auth")
		}
		// For SAS, append token to URL as query string
		sasURL := accountURL
		if sasURL[len(sasURL)-1] == '/' {
			sasURL = sasURL[:len(sasURL)-1]
		}
		// Add ? if token doesn't start with it
		if len(v) > 0 && v[0] != '?' {
			sasURL += "?"
		}
		return azblob.NewClientWithNoCredential(sasURL+v, nil)

	default:
		return nil, fmt.Errorf("unsupported credential type: %T", cred)
	}
}
