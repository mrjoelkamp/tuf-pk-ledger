package ledger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/mrjoelkamp/opkl-updater/log"
)

func parseProviderURI(providerURI string) (*url.URL, error) {
	parsedURI, err := url.ParseRequestURI(providerURI)
	if err != nil {
		return nil, err
	}
	if parsedURI.Scheme != "https" || !parsedURI.IsAbs() {
		return nil, fmt.Errorf("Provider URI [%s] is not valid", providerURI)
	}
	log.Info("parsed uri", "scheme", parsedURI.Scheme, "host", parsedURI.Host, "path", parsedURI.Path)
	return parsedURI, nil
}

func createNewProviderIndexEntry(parsedURI *url.URL, opIdx IssIndex) (*IssIndexItem, error) {
	issuer := stripTrailingSlash(parsedURI.String())
	opIdxItem := new(IssIndexItem)
	opIdxItem.Path = filepath.Join(LedgerPath, parsedURI.Host, LedgerIndexFilename)

	// add to issuer index
	opIdx.Issuers[issuer] = *opIdxItem
	err := writeJSONFile(filepath.Join(LedgerPath, IssuerIndexFilename), opIdx)
	if err != nil {
		return nil, err
	}
	log.Info("Created new provider index.", "issuer", issuer, "path", opIdxItem.Path)
	return opIdxItem, nil
}

func getOpenIDConfiguration(url *url.URL) (map[string]interface{}, error) {
	// Construct the URL for the OpenID configuration
	configURL := url.JoinPath(OidcDiscoveryPath)

	// Make a GET request to the OpenID configuration endpoint
	resp, err := http.Get(configURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check if the response status code is OK (200)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to retrieve OpenID configuration. Status code: %d", resp.StatusCode)
	}

	// Parse the JSON response
	var config map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func getJWKS(providerURI *url.URL) (*JWKS, int64, error) {
	// query openid-configuration for jwks_uri
	cfgOIDC, err := getOpenIDConfiguration(providerURI)
	if err != nil {
		return nil, 0, err
	}
	jwksURI, ok := cfgOIDC[JwksKey].(string)
	if !ok {
		log.Error("Key not found in configuration", "key", jwksURI)
	}
	parsedJwksURI, err := url.ParseRequestURI(jwksURI)

	// Make a GET request to the JWKS endpoint
	resp, err := http.Get(parsedJwksURI.String())
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	timestamp := time.Now().Unix()

	// Check if the response status code is OK (200)
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("Failed to retrieve JWKS. Status code: %d", resp.StatusCode)
	}

	// Parse the JSON response
	jwks := new(JWKS)
	err = json.NewDecoder(resp.Body).Decode(jwks)
	if err != nil {
		return nil, 0, err
	}
	err = jwks.Unmarshal() // TODO - implement this as json unmarshal interface
	if err != nil {
		return nil, 0, err
	}
	return jwks, timestamp, nil
}

func getIssuerIndex(filePath string) (IssIndex, error) {
	opIdx := IssIndex{Issuers: make(map[string]IssIndexItem)}
	exists, err := fileExists(filePath)
	if err != nil {
		return opIdx, err
	}
	if exists {
		err := readJSONFile(filePath, &opIdx)
		if err != nil {
			return IssIndex{}, err
		}
		return opIdx, nil
	}
	return opIdx, nil
}

func lookupProvider(parsedURI *url.URL, index IssIndex) *IssIndexItem {
	nomarlizedURI := stripTrailingSlash(parsedURI.String())
	iss, ok := index.Issuers[nomarlizedURI]
	if ok {
		return &iss
	}
	return nil
}
