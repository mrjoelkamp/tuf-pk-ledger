package ledger

import "encoding/json"

type JWK struct {
	RawJSON map[string]interface{} `json:"-"`
	Mod     string                 `json:"n"`   // required for RSA
	Exp     string                 `json:"e"`   // required for RSA
	Kty     string                 `json:"kty"` // TODO validate as RSA required for OPKL
	Use     string                 `json:"use,omitempty"`
	KeyOps  []string               `json:"key_ops,omitempty"`
	Alg     string                 `json:"alg,omitempty"`
	Kid     string                 `json:"kid"` // kid is OPTIONAL in RFC 7517 but currently required for OPKL
	X5u     string                 `json:"x5u,omitempty"`
	X5c     []string               `json:"x5c,omitempty"`
	X5t     string                 `json:"x5t,omitempty"`
	X5tS256 string                 `json:"x5t#S256,omitempty"`
}

type JWKS struct {
	Keys    []JWK                    `json:"-"`
	RawJSON []map[string]interface{} `json:"keys"`
}

func (jwks *JWKS) Unmarshal() error {
	// Unmarshal keys into []JWK while preserving original JWK JSON
	for _, jwk := range jwks.RawJSON {
		var obj JWK
		jsonString, err := json.Marshal(jwk)
		err = json.Unmarshal(jsonString, &obj)
		if err != nil {
			return err
		}
		obj.RawJSON = jwk
		jwks.Keys = append(jwks.Keys, obj)
	}
	return nil
}

type PklFile struct {
	Jwk map[string]interface{} `json:"jwk"`
	Nbf *int64                 `json:"nbf"`
	Exp *int64                 `json:"exp"`
}

type PklIndexItem struct {
	Status string `json:"status"` // TODO validate it is one of active|archived|revoked
	Path   string `json:"path"`
}

type PklIndex struct {
	Items map[string]PklIndexItem `json:"pkl"`
}

type IssIndexItem struct {
	Path string `json:"path"`
}

type IssIndex struct {
	Issuers map[string]IssIndexItem `json:"issuers"`
}
