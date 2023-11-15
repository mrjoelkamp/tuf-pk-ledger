package ledger

import (
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

	// parse and validate input
	parsedURI, err := parseProviderURI(providerURI)
	if err != nil {
		return err
	}

	// get provider index
	opIdx, err := getIssuerIndex(filepath.Join(LedgerPath, IssuerIndexFilename))
	if err != nil {
		return err
	}

	// lookup or create provider index item for this provider
	opIdxItem := lookupProvider(parsedURI, opIdx)

	// create new entry if provider not found
	if opIdxItem == nil {
		opIdxItem, err = createNewProviderIndexEntry(parsedURI, opIdx)
		if err != nil {
			return err
		}
	}

	// get key ledger index
	pklIdx, err := getPklIndex(opIdxItem.Path)
	if err != nil {
		return err
	}

	// get active keys to detect key rotation
	expectedActiveJWKs := make(map[string]PklIndexItem)
	for id, jwkIdx := range pklIdx.Items {
		if jwkIdx.Status == "active" {
			expectedActiveJWKs[id] = jwkIdx
		}
	}

	// query jwks_uri and record time
	jwks, timestamp, err := getJWKS(parsedURI)
	if err != nil {
		return err
	}
	if cfg.GetString("loglevel") == "debug" {
		json, err := jsonStructToString(jwks)
		if err != nil {
			return err
		}
		log.Debugf(json)
		log.Debugf("timestamp=%d", timestamp)
	}

	// update JWK ledger files based on JWKS response from OP
	pklIdxUpdated := false
	for _, jwk := range jwks.Keys {
		// check if JWK already exists in ledger
		if jwkInLedger(jwk, pklIdx) {
			// reconcile active jwk to detect key rotation (active key not in JWKS response)
			reconcileExpectedJWK(jwk, expectedActiveJWKs)

			// TODO check configuration parameter for fail-safe updates
			// then update exp time based on JWKS query timestamp if true
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
	pklIdxUpdated, err = detectRotatedJWK(pklIdx, expectedActiveJWKs, timestamp)
	if err != nil {
		return err
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

func detectRotatedJWK(pklIdx PklIndex, expectedActiveJWKs map[string]PklIndexItem, timestamp int64) (bool, error) {
	if len(expectedActiveJWKs) > 0 {
		log.Infof("remaining active JWKs: %d", len(expectedActiveJWKs))
		// key was rotated set exp and update ledger index
		for id, rotatedJWK := range expectedActiveJWKs {
			// set exp for jwk
			var jwk PklFile
			err := readJSONFile(rotatedJWK.Path, &jwk)
			if err != nil {
				return false, err
			}
			jwk.Exp = &timestamp
			err = writeJSONFile(rotatedJWK.Path, jwk)
			if err != nil {
				return false, err
			}

			// update ledger index status
			indexItem, ok := pklIdx.Items[id]
			if ok {
				indexItem.Status = StatusArchived
				pklIdx.Items[id] = indexItem
			}
			return true, nil
		}
	}
	return false, nil
}

func reconcileExpectedJWK(jwk JWK, activeJWKs map[string]PklIndexItem) {
	_, ok := activeJWKs[jwk.Kid]
	if ok {
		delete(activeJWKs, jwk.Kid)
	}
}

func jwkInLedger(jwk JWK, idx PklIndex) bool {
	_, ok := idx.Items[jwk.Kid]
	return ok
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
