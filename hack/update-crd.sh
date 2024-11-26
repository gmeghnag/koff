#!/bin/bash
set -euo pipefail

###
#
# To add a new CRD:
#  - add the relevant module to pkg/dependencymagnet/doc.go
#  - run `go mod vendor`
#  - add an `install_crd` to the list below here
#  - run `go generate ./...`
#
###

function install_crd {
  local SRC="$1"
  local DST="$2"
  if [[ -e "$SRC" ]]
  then
    if [[ -e "$DST" ]]
    then
      if ! diff -Naup "$SRC" "$DST"; then
        cp "$SRC" "$DST"
        echo "updated CRD: $SRC => $DST"
      else
        echo "skipped CRD that is already up to date: $DST"
      fi
    else
      cp "$SRC" "$DST"
      echo "updated CRD: $SRC => $DST"
    fi
  else
    if [[ -e "$DST" ]]
    then
      rm "$DST"
      echo "removed CRD that not vendored: $DST"
    else
      echo "skipped CRD that is not vendored: $SRC"
    fi
  fi
}

# Can't rely on associative arrays for old Bash versions (e.g. OSX)

shopt -s extglob
install_crd \
  vendor/github.com/openshift/api/machineconfiguration/v1/zz_generated.crd-manifests/0000_80_machine-config_01_machineconfigs.crd.yaml \
  "manifests/machineconfigs-custom-resource-definition.yaml"

install_crd \
  vendor/github.com/openshift/api/config/v1/zz_generated.crd-manifests/0000_00_cluster-version-operator_01_clusterversions-Default.crd.yaml \
  "manifests/clusterversions-custom-resource-definition.yaml"

install_crd \
  vendor/github.com/openshift/api/config/v1/zz_generated.crd-manifests/0000_00_cluster-version-operator_01_clusteroperators.crd.yaml \
  "manifests/clusteroperators-custom-resource-definition.yaml"
