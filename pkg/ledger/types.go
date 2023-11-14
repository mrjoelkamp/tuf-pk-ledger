package ledger

type JWK struct {
	RawJSON string   `json:"-"`
	Mod     string   `json:"n"`   // required for RSA
	Exp     string   `json:"e"`   // required for RSA
	Kty     string   `json:"kty"` // TODO validate as RSA required for OPKL
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
