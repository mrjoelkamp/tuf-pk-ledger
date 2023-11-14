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
		issuer := stripTrailingSlash(parsedURI.String())
		opIdxItem = IssIndexItem{
			Path: filepath.Join(LedgerPath, parsedURI.Host, LedgerIndexFilename),
		}
		// add to issuer index
		opIdx.Issuers[issuer] = opIdxItem
		err = writeJSONFile(filepath.Join(LedgerPath, IssuerIndexFilename), opIdx)
		if err != nil {
			return err
		}
		log.Infof("Created new provider index. issuer=%s path=%s", issuer, opIdxItem.Path)
	}

	// get key ledger index
	pklIdx, err := getPklIndex(opIdxItem.Path)
	if err != nil {
		return err
	}

	// get active keys to detect key rotation
	remainingActiveJWKs := make(map[string]PklIndexItem)
	for id, jwkIdx := range pklIdx.Items {
		if jwkIdx.Status == "active" {
			remainingActiveJWKs[id] = jwkIdx
		}
	}

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
			// reconcile active jwk to detect key rotation (active key not in JWKS response)
			reconcileActiveJWK(jwk, remainingActiveJWKs)

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
			Status: StatusActive,
			Path:   ledgerFilePath,
		}
		pklIdx.Items[jwk.Kid] = newPklIndexItem
		pklIdxUpdated = true
	}
	// detect rotated keys
	if len(remainingActiveJWKs) > 0 {
		log.Infof("remaining active JWKs: %d", len(remainingActiveJWKs))
		// key was rotated set exp and update ledger index
		for id, rotatedJWK := range remainingActiveJWKs {
			// set exp for jwk
			var jwk PklFile
			err = readJSONFile(rotatedJWK.Path, &jwk)
			if err != nil {
				return err
			}
			jwk.Exp = &timestamp
			err = writeJSONFile(rotatedJWK.Path, jwk)
			if err != nil {
				return err
			}

			// update ledger index status
			indexItem, ok := pklIdx.Items[id]
			if ok {
				indexItem.Status = StatusArchived
				pklIdx.Items[id] = indexItem
			}
			pklIdxUpdated = true
		}
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

func reconcileActiveJWK(jwk JWK, activeJWKs map[string]PklIndexItem) {
	_, ok := activeJWKs[jwk.Kid]
	if ok {
		delete(activeJWKs, jwk.Kid)
	}
}

func jwkInLedger(jwk JWK, idx PklIndex) bool {
	_, ok := idx.Items[jwk.Kid]
	if ok {
		return true
	}
	return false
}

func getPklIndex(filePath string) (PklIndex, error) {
	pklIdx := PklIndex{make(map[string]PklIndexItem)}
	exists, err := fileExists(filePath)
	if err != nil {
		return pklIdx, err
	}
	if exists {
		err := readJSONFile(filePath, &pklIdx)
		if err != nil {
			return pklIdx, err
		}
		return pklIdx, nil
	}
	return pklIdx, nil
}
