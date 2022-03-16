#!/usr/bin/env bash
set -eu

FS_NAME="alpinefs"
FS_MOUNT="/tmp/${FS_NAME}"
DOCKER_TAG="${FS_NAME}:latest"

pushd actionloop && make && popd

docker build . --tag "${DOCKER_TAG}"

dd if=/dev/zero of="${FS_NAME}.ext4" bs=1M count=256
mkfs.ext4 "${FS_NAME}.ext4"

sudo umount "${FS_MOUNT}" || true
sudo rm -rf "${FS_MOUNT}"
mkdir "${FS_MOUNT}"
sudo mount "${FS_NAME}.ext4" "${FS_MOUNT}"


docker run --rm -it \
  -v "${FS_MOUNT}:/${FS_NAME}" \
  --entrypoint copy-fs.sh \
  "${DOCKER_TAG}" "/${FS_NAME}"

sudo umount "${FS_MOUNT}"





