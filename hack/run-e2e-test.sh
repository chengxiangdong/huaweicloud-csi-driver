#!/usr/bin/env bash

# Copyright 2022 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

function usage() {
  echo "This script run E2E tests by ginkgo."
  echo "      Usage: hack/run-e2e-test.sh <MODULE_NAME>"
  echo "    Example: hack/run-e2e-test.sh SFS"
  echo
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

labels=${1}
if [[ -z "${labels}" ]]; then
  usage
  exit 1
fi

if [[ "${labels}" = "NONE" ]]; then
  echo "No test module is specified, all E2E tests are skipped."
  echo
  exit 0
fi

REPO_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

# create cluster
export CLUSTER_NAME=${CLUSTER_NAME:-"k8s-cluster"}
${REPO_ROOT}/hack/create-cluster.sh ${CLUSTER_NAME}

ARTIFACTS_PATH=${ARTIFACTS_PATH:-"${HOME}/e2e-logs"}
mkdir -p "$ARTIFACTS_PATH"

GO111MODULE=on go install github.com/onsi/ginkgo/v2/ginkgo@v2.2.0
GOPATH=$(go env GOPATH | awk -F ':' '{print $1}')
export PATH=$PATH:$GOPATH/bin

# pre run e2e
"${REPO_ROOT}"/hack/pre-run-e2e.sh ${CLUSTER_NAME}

# run e2e test
set +e
ginkgo -v --race --trace --fail-fast -p --randomize-all --label-filter=${labels}  ./test/e2e/
TESTING_RESULT=$?

# Collect logs
echo "Collect logs to $ARTIFACTS_PATH..."

echo "Collecting $CLUSTER_NAME logs..."
mkdir -p "$ARTIFACTS_PATH/$CLUSTER_NAME"
kind export logs --name="$CLUSTER_NAME" "$ARTIFACTS_PATH/$CLUSTER_NAME"

echo "Collected logs at $ARTIFACTS_PATH:"
ls -al "$ARTIFACTS_PATH"

# post run e2e
"${REPO_ROOT}"/hack/post-run-sfsturbo-e2e.sh

# delete cluster
${REPO_ROOT}/hack/delete-cluster.sh ${CLUSTER_NAME}

exit $TESTING_RESULT
