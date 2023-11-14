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
	log.Infof("[parsed uri] scheme=%s host=%s path=%s", parsedURI.Scheme, parsedURI.Host, parsedURI.Path)
	return parsedURI, nil
}

func createNewProviderIndexEntry(parsedURI *url.URL, opIdx IssIndex) (IssIndexItem, error) {
	issuer := stripTrailingSlash(parsedURI.String())
	opIdxItem := IssIndexItem{
		Path: filepath.Join(LedgerPath, parsedURI.Host, LedgerIndexFilename),
	}
	// add to issuer index
	opIdx.Issuers[issuer] = opIdxItem
	err := writeJSONFile(filepath.Join(LedgerPath, IssuerIndexFilename), opIdx)
	if err != nil {
		return IssIndexItem{}, err
	}
	log.Infof("Created new provider index. issuer=%s path=%s", issuer, opIdxItem.Path)
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

func getJWKS(url *url.URL) (JWKS, int64, error) {
	// Make a GET request to the JWKS endpoint
	resp, err := http.Get(url.String())
	if err != nil {
		return JWKS{}, 0, err
	}
	defer resp.Body.Close()
	timestamp := time.Now().Unix()

	// Check if the response status code is OK (200)
	if resp.StatusCode != http.StatusOK {
		return JWKS{}, 0, fmt.Errorf("Failed to retrieve JWKS. Status code: %d", resp.StatusCode)
	}

	// Parse the JSON response
	var jwks JWKS
	err = json.NewDecoder(resp.Body).Decode(&jwks)
	if err != nil {
		return JWKS{}, 0, err
	}
	err = jwks.Unmarshal()
	if err != nil {
		return JWKS{}, 0, err
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

func lookupProvider(parsedURI *url.URL, index IssIndex) (IssIndexItem, error) {
	nomarlizedURI := stripTrailingSlash(parsedURI.String())
	iss, ok := index.Issuers[nomarlizedURI]
	if ok {
		return iss, nil
	}
	return IssIndexItem{}, nil
}
