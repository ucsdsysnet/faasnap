#!/usr/bin/env sh

# Then, copy the newly configured system to the rootfs image:
for d in bin etc lib root sbin usr opt; do tar c "/$d" | tar x -C ${1}; done

# The above command may trigger the following message:
# tar: Removing leading "/" from member names
# However, this is just a warning, so you should be able to
# proceed with the setup process.

for dir in dev proc run sys var var/run; do mkdir ${1}/${dir}; done

# All done, exit docker shell.
