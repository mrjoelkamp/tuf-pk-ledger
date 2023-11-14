package ledger

import (
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/mrjoelkamp/opkl-updater/config"
	"github.com/mrjoelkamp/opkl-updater/log"
)

const (
	OidcDiscoveryPath   = "/.well-known/openid-configuration"
	JwksKey             = "jwks_uri"
	LedgerPath          = "targets/opkl"
	IssuerIndexFilename = "issuers.json"
	LedgerIndexFilename = "pkl.json"
	LedgerFileExt       = ".json"
	StatusActive        = "active"
	StatusArchived      = "archived"
	StatusRevoked       = "revoked"
)

func Update(providerURI string) error {
	cfg := config.Config()

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

	// lookup or create provider index item
	opIdxItem, err := lookupProvider(parsedURI, opIdx)
	if err != nil {
		return err
	}
	log.Debugf(opIdxItem.Path)

	// create new entry if provider not found
	if opIdxItem.Path == "" {
		opIdxItem = IssIndexItem{
			Issuer: stripTrailingSlash(parsedURI.String()),
			Path:   filepath.Join(LedgerPath, parsedURI.Host, LedgerIndexFilename),
		}
		// append to issuer index
		opIdx.Issuers = append(opIdx.Issuers, opIdxItem)
		err = writeJSONFile(filepath.Join(LedgerPath, IssuerIndexFilename), opIdx)
		if err != nil {
			return err
		}
		log.Infof("Created new provider index. issuer=%s path=%s", opIdxItem.Issuer, opIdxItem.Path)
	}

	// get key ledger index
	pklIdx, err := getPklIndex(opIdxItem.Path)
	if err != nil {
		return err
	}

	// TODO get active keys to detect key rotation
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
	pklIdxUpdated := false
	for _, jwk := range jwks.Keys {
		// check if JWK already exists in ledger
		if jwkInLedger(jwk, pklIdx) {
			// TODO check configuration parameter for fail-safe updates
			// then update exp time based on JWKS query timestamp if ture
			continue
		}
		// write new jwk ledger file
		pklID := hashString(jwk.Kid)
		newPklFile := PklFile{
			Jwk: jwk.RawJSON,
			Nbf: &timestamp,
			Exp: nil,
		}
		ledgerFilePath := filepath.Join(LedgerPath, parsedURI.Host, pklID+LedgerFileExt)
		err = writeJSONFile(ledgerFilePath, newPklFile)
		if err != nil {
			return err
		}

		// add JWK to ledger index
		newPklIndexItem := PklIndexItem{
			Kid:    jwk.Kid,
			Status: StatusActive,
			Path:   ledgerFilePath,
		}
		pklIdx.Items = append(pklIdx.Items, newPklIndexItem)
		pklIdxUpdated = true

		// TODO detect key rotation (active key not in JWKS response)
	}
	// write ledger index updates if modified
	if pklIdxUpdated {
		err = writeJSONFile(opIdxItem.Path, pklIdx)
		if err != nil {
			return err
		}
	}

	return nil
}

func jwkInLedger(jwk JWK, idx PklIndex) bool {
	for _, item := range idx.Items {
		if item.Kid == jwk.Kid {
			return true
		}
	}
	return false
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
