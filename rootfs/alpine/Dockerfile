FROM amd64/alpine:3.13

RUN apk add --no-cache openrc util-linux syslinux

RUN ln -s agetty /etc/init.d/agetty.ttyS0 && \
    echo ttyS0 > /etc/securetty && \
    rc-update add agetty.ttyS0 default && \
    rc-update add devfs boot && \
    rc-update add procfs boot && \
    rc-update add sysfs boot && \
    rc-update add networking boot && \
    echo "root:root" | chpasswd

ADD interfaces /etc/network/interfaces
ADD action-server /etc/init.d/action-server
ADD actionloop/target/x86_64-unknown-linux-musl/release/actionloop /usr/bin/actionloop

RUN chmod 755 /etc/init.d/action-server && \
    chmod +x /etc/init.d/action-server && \
    rc-update add action-server

ADD copy-fs.sh /usr/bin/
