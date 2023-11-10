#!/bin/bash
set -xeo pipefail

# Set number of retries if commit fails
RETRIES=5
# Set time to wait before retry attempts
WAIT_SECONDS=3

git config user.name opkl-update-agent
git config user.email mrjoelkamp@gmail.com
git add .
git commit -m "${COMMIT_MESSAGE}" || err=$?
if [ -z "${err}" ]; then
    for (( i=1; i<=$RETRIES; i++ )); do
        git pull --rebase
        git push && break || sleep $WAIT_SECONDS
    done
fi
