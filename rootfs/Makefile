.PHONY: all clean

DIST=debian
DEBIAN_VERSION=buster # buster-slim
GUESTFILES := $(wildcard guest/*)

all:

debian-base-rootfs.ext4: scripts/create-debian-base-rootfs.sh
	scripts/create-debian-base-rootfs.sh $(DEBIAN_VERSION) # install base image

debian-provisioned-rootfs.ext4: debian-base-rootfs.ext4 scripts/provision-debian-rootfs.sh scripts/setup-debian-rootfs.sh
	scripts/provision-debian-rootfs.sh # install dependencies, sys config

debian-rootfs.ext4: debian-provisioned-rootfs.ext4 scripts/ready-debian-rootfs.sh $(GUESTFILES)
	scripts/ready-debian-rootfs.sh # install functions

alpinefs.ext4: alpine/alpine-rootfs.sh
	cd alpine/ && ./alpine-rootfs.sh
	mv alpine/alpinefs.ext4 .


clean:
	rm -rf *.ext4

