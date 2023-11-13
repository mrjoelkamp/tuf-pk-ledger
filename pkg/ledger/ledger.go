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

type JWK struct {
	RawJSON string   `json:"-"`
	Mod     string   `json:"n"`
	Exp     string   `json:"e"`
	Kty     string   `json:"kty"`
	Use     string   `json:"use,omitempty"`
	KeyOps  []string `json:"key_ops,omitempty"`
	Alg     string   `json:"alg,omitempty"`
	Kid     string   `json:"kid"` // kid is OPTIONAL in RFC 7517 but currently required for OPKL
	X5u     string   `json:"x5u,omitempty"`
	X5c     []string `json:"x5c,omitempty"`
	X5t     string   `json:"x5t,omitempty"`
	X5tS256 string   `json:"x5t#S256,omitempty"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type PklFile struct {
	Jwk map[string]interface{} `json:"jwk"`
	Nbf int64                  `json:"nbf"`
	Exp int64                  `json:"exp"`
}

type PklIndexItem struct {
	Kid    string `json:"kid"`
	Status string `json:"status"` // TODO validate it is one of active|archived|revoked
	Path   string `json:"path"`
}

type PklIndex struct {
	Items []PklIndexItem `json:"pkl"`
}

type IssIndexItem struct {
	Issuer string `json:"iss"`
	Path   string `json:"path"`
}

type IssIndex struct {
	Issuers []IssIndexItem `json:"issuers"`
}

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
	log.Debugf("[parsed uri] scheme=%s host=%s path=%s", parsedURI.Scheme, parsedURI.Host, parsedURI.Path)

	// get provider index
	opIdx, err := getIssuerIndex(filepath.Join(LedgerPath, LedgerIndexFilename))
	if err != nil {
		return err
	}

	// lookup or create provider index item
	opIdxItem, err := lookupProvider(providerURI, opIdx)
	if err != nil {
		return err
	}
	log.Debugf(opIdxItem.Path)

	// TODO get ledger index
	// TODO get active keys

	// query openid-configuration
	cfgOIDC, err := getOpenIDConfiguration(parsedURI)
	if err != nil {
		return err
	}
	jwksURI, ok := cfgOIDC[JwksKey].(string)
	if !ok {
		log.Errorf("Key '%s' not found in configuration", jwksURI)
	}
	log.Debugf("%s=%s", JwksKey, jwksURI)
	parsedJwksURI, err := url.ParseRequestURI(jwksURI)

	// query JWKS URI and record time
	jwks, timestamp, err := getJWKS(parsedJwksURI)
	if err != nil {
		return err
	}
	if cfg.GetString("loglevel") == "debug" {
		err = prettyPrintJsonStruct(jwks)
		if err != nil {
			return err
		}
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

func prettyPrintJsonStruct(v any) error {
	stringData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	log.Debugf(string(stringData))
	return nil
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
		err := readJSONFile(filepath.Join(LedgerPath, IssuerIndexFilename), opIdx)
		if err != nil {
			return IssIndex{}, nil
		}
		return opIdx, nil
	}
	return opIdx, nil
}

func lookupProvider(provider string, index IssIndex) (IssIndexItem, error) {
	for _, iss := range index.Issuers {
		if iss.Issuer == provider {
			return iss, nil
		}
	}
	return IssIndexItem{}, nil
}
