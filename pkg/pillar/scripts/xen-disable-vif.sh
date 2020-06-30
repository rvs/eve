#!/bin/sh
set -ex

if [ $# -ne 1 ]; then
   echo "Usage: $0 <domain ID>"
   exit 1
fi

xenstore ls -f "/local/domain/$1/device/vif" | awk '/\/feature-/ { print $1;}' | while read node; do
  xenstore write "$node" 0
  # /local/domain/X/device/vif/Y/feature-Z -> /local/domain/0/backend/vif/X/Y/feature-Z
  xenstore write "/local/domain/0/backend/vif/$1/$(echo $node | sed -e 's#^.*device/vif/##')" 0
done
