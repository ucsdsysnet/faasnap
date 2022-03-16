#! /usr/bin/env bash

set -ex

IN=debian-provisioned-rootfs.ext4
OUT=debian-rootfs.ext4
TMPOUT=.$OUT

sudo umount ./mountpoint || true
sudo rm -rf ./mountpoint
mkdir -p ./mountpoint
cp $IN $TMPOUT

sudo mount $TMPOUT mountpoint
sudo mkdir mountpoint/app
sudo cp guest/* mountpoint/app/

sudo umount mountpoint
mv $TMPOUT $OUT
