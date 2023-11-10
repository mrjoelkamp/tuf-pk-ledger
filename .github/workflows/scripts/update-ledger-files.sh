#!/bin/bash
set -xeo pipefail

# Use ENV for OP_URI or default to github
OP_URI="${OP_URI:-"https://token.actions.githubusercontent.com"}"
OP_URI_NORMALIZED="$($OP_URI%/)"
OP_ID=$(echo "$OP_URI_NORMALIZED" | cut -d'/' -f3)
REPO_PATH=$(pwd)
OIDC_CONFIG_URI=${OP_URI_NORMALIZED}/.well-known/openid-configuration
LEDGER_PATH=targets/opkl/${OP_ID}
PKL_INDEX=targets/opkl/${OP_ID}/pkl.json

# Get OIDC provider's openid-configuration
config_response=$(curl -s $OIDC_CONFIG_URI)

# Parse jwks_uri from openid-configuration
if [ $? -eq 0 ]; then
    jwks_uri=$(echo "$config_response" | jq -r '.jwks_uri')
else
    echo "error: failed to retrieve data from ${OIDC_CONFIG_URI}"
    exit 1
fi

# Get OIDC provider's JWKS
jwks_response=$(curl -s ${jwks_uri})
request_time=$(date +%s)

# Parse JWKS
if [ $? -eq 0 ]; then
    keys=$(echo "$jwks_response" | jq -r '.keys')
else
    echo "error: failed to retrieve data from ${jwks_uri}"
    exit 1
fi

# Write JWK ledger files
create_jwk() {
        mkdir -p "$(dirname "$filepath")"
        touch $filepath
        jwk_json=$(cat <<-EOF 
{
    "jwk": $1,
    "nbf": $request_time,
    "exp": null
}
EOF
        )
        echo "$jwk_json" > "$filepath"
        echo "wrote ${filepath}"
}

update_pkl_index() {
    if [ -e "$REPO_PATH/$PKL_INDEX" ]; then
        # update pkl index
        echo "update pkl index"
        new_jwk=$(cat <<-EOF
{"kid":"$1","path":"$2"}
EOF
        )
        jq --argjson new_jwk "$new_jwk" '.pkl += [$new_jwk]' "$REPO_PATH/$PKL_INDEX" > new_pkl.json && mv new_pkl.json "$REPO_PATH/$PKL_INDEX"
    else
        # create pkl index
        touch $REPO_PATH/$PKL_INDEX
        pkl_json=$(cat <<-EOF
{
    "pkl": [
        {"kid":"$1","path":"$2"}
        ]
} 
EOF
    )
        echo "creating $PKL_INDEX"
        echo "$pkl_json" > "$PKL_INDEX"
    fi
}

# Iterate over keys to create or update ledger files
for jwk in $(echo "$keys" | jq -c '.[]'); do
    kid=$(echo "$jwk" | jq -r '.kid')
    filepath=${REPO_PATH}/${LEDGER_PATH}/${kid}.json
    if [ -e "$filepath" ]; then
        # for fail-safe implementation set 'exp' to request_time
        echo "jwk ${kid} already exists"
    else
        create_jwk $jwk
        update_pkl_index $kid "/${LEDGER_PATH}/${kid}.json"
    fi
done
