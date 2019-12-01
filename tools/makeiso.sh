#!/bin/sh

MKIMAGE_TAG="$(linuxkit pkg show-tag pkg/mkimage-iso-efi)"
PARTS="$(cd $1 && pwd)"
ISO="$(cd $(dirname $2) && pwd)/$(basename $2)"

if [ ! -d "$PARTS" -o $# -ne 2 ]; then
   echo "Usage: $0 <input dir> <output iso image file>"
   exit 1
fi

touch "$PARTS/../boot.img" "$2"
docker run -t -v "$PARTS:/mnt" -v "$PARTS/../boot.img:/mnt/boot.img" -v "$ISO:/output.iso" -i ${MKIMAGE_TAG}
