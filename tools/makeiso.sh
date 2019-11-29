#!/bin/sh
# Usage:
#
#     ./makeiso.sh > <output.iso>
#
MKIMAGE_TAG="$(linuxkit pkg show-tag pkg/mkimage-iso-efi)"

docker run -i ${MKIMAGE_TAG} > $1
