package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/mrjoelkamp/opkl-updater/config"
	"github.com/mrjoelkamp/opkl-updater/log"
)

const (
	OidcDiscoveryPath   = "/.well-known/openid-configuration"
	JwksKey             = "jwks_uri"
	LedgerPath          = "targets/opkl"
	IssuerIndexFilename = "issuers.json"
	LedgerIndexFilename = "pkl.json"
)

var validate *validator.Validate

func Update(providerURI string) error {
	cfg := config.Config()
	validate = validator.New(validator.WithRequiredStructEnabled())

	// validate input
	parsedURI, err := url.ParseRequestURI(providerURI)
	if err != nil {
		return err
	}
	if parsedURI.Scheme != "https" || !parsedURI.IsAbs() {
		return fmt.Errorf("Provider URI [%s] is not valid", providerURI)
	}
	log.Infof("[parsed uri] scheme=%s host=%s path=%s", parsedURI.Scheme, parsedURI.Host, parsedURI.Path)

	// get provider index
	opIdx, err := getIssuerIndex(filepath.Join(LedgerPath, IssuerIndexFilename))
	if err != nil {
		return err
	}
	err = validate.Struct(opIdx)
	if err != nil {
		return err
	}

	// lookup or create provider index item
	opIdxItem, err := lookupProvider(parsedURI, opIdx)
	if err != nil {
		return err
	}
	log.Debugf(opIdxItem.Path)
	// create new entry if provider not found
	if opIdxItem.Path == "" {
		opIdxItem := IssIndexItem{
			Issuer: stripTrailingSlash(parsedURI.String()),
			Path:   filepath.Join(LedgerPath, parsedURI.Host, LedgerIndexFilename),
		}
		// append to issuer index
		opIdx.Issuers = append(opIdx.Issuers, opIdxItem)
		data, err := jsonStructToString(opIdx)
		if err != nil {
			return err
		}
		createFile(filepath.Join(LedgerPath, IssuerIndexFilename), data)
		log.Infof("Created new provider index. issuer=%s path=%s", opIdxItem.Issuer, opIdxItem.Path)
	}

	// get key ledger index
	// pklIdx, err := getPklIndex(opIdxItem.Path)
	// if err != nil {
	// 	return err
	// }

	// get active keys
	// var activeJWKs []JWK
	// for _, jwkIdx := range pklIdx.Items {
	// 	if jwkIdx.Status == "active" {

	// 		activeJWKs = append(activeJWKs, jwkIdx.Path)
	// 	}
	// }

	// query openid-configuration
	cfgOIDC, err := getOpenIDConfiguration(parsedURI)
	if err != nil {
		return err
	}
	jwksURI, ok := cfgOIDC[JwksKey].(string)
	if !ok {
		log.Errorf("Key '%s' not found in configuration", jwksURI)
	}
	parsedJwksURI, err := url.ParseRequestURI(jwksURI)

	// query JWKS URI and record time
	jwks, timestamp, err := getJWKS(parsedJwksURI)
	if err != nil {
		return err
	}
	if cfg.GetString("loglevel") == "debug" {
		json, err := jsonStructToString(jwks)
		if err != nil {
			return err
		}
		log.Debugf(json)
	}
	log.Debugf("timestamp=%d", timestamp)

	// for each JWK in JWKS
	// check if JWK already exists in ledger
	// if JWK doesn't exist create ledger file
	// if JWK does exist check configuration parameter for fail-safe updates
	// update ledger index

	// detect key rotation (active key not in JWKS response)

	return nil
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

	return jwks, timestamp, nil
}

func jsonStructToString(v any) (string, error) {
	stringData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(stringData), nil
}

func hashString(input string) string {
	s256 := sha256.New()
	s256.Write([]byte(input))
	hashSum := s256.Sum(nil)
	hash := hex.EncodeToString(hashSum)
	return hash
}

func readJSONFile(path string, v any) error {
	// Read JSON file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Unmarshal JSON data into the struct
	err = json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	return nil
}

func writeJSONFile(path string, v any) error {
	// Marshal the struct into JSON format
	data, err := json.MarshalIndent(&v, "", "  ")
	if err != nil {
		return err
	}

	// Write JSON data to a file
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func fileExists(filePath string) (bool, error) {
	// Check if the file or path exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File or path does not exist
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		// File or path already exists
		return true, nil
	}
}

func createFile(filePath string, data string) error {
	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write data to the file
	_, err = file.WriteString(data)
	if err != nil {
		return err
	}

	return nil
}

func getIssuerIndex(filePath string) (IssIndex, error) {
	var opIdx IssIndex
	exists, err := fileExists(filePath)
	if err != nil {
		return IssIndex{}, err
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

func getPklIndex(filePath string) (PklIndex, error) {
	var pklIdx PklIndex
	exists, err := fileExists(filePath)
	if err != nil {
		return PklIndex{}, err
	}
	if exists {
		err := readJSONFile(filePath, &pklIdx)
		if err != nil {
			return PklIndex{}, err
		}
		return pklIdx, nil
	}
	return pklIdx, nil
}

func stripTrailingSlash(url string) string {
	return strings.TrimSuffix(url, "/")
}

func lookupProvider(parsedURI *url.URL, index IssIndex) (IssIndexItem, error) {
	nomarlizedURI := stripTrailingSlash(parsedURI.String())
	for _, iss := range index.Issuers {
		if iss.Issuer == nomarlizedURI {
			return iss, nil
		}
	}
	return IssIndexItem{}, nil
}
