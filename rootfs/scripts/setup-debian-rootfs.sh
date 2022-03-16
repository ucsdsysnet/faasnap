#! /usr/bin/env bash
set -ex

echo "debian" > /etc/hostname
echo root:rootroot | chpasswd

apt install -y libhdf5-dev gpg wget libblas3 liblapack3 liblapack-dev libblas-dev gfortran
wget https://github.com/Kitware/CMake/releases/download/v3.22.2/cmake-3.22.2-linux-x86_64.tar.gz -O /opt/cmake-3.22.2-linux-x86_64.tar.gz
pushd /opt
tar xzvf cmake-3.22.2-linux-x86_64.tar.gz
export PATH=$PATH:/opt/cmake-3.22.2-linux-x86_64/bin/
popd
# apt install -y tcpdump build-essential pkg-config python3-setuptools python-dev python3-dev gcc libpq-dev python-pip python3-dev python3-pip python3-venv python3-wheel
pip3 install wheel six scikit-learn==0.23.0 flask redis pillow pyaes Chameleon pandas tensorflow grpcio==1.36.1 igraph torch==1.10.2 torchvision==0.11.3 # torchaudio==0.10.2+cpu #opencv-contrib-python 

mkdir -p /etc/systemd/system/serial-getty@ttyS0.service.d/
cat <<EOF > /etc/systemd/system/serial-getty@ttyS0.service.d/autologin.conf
[Service]
ExecStart=
ExecStart=-/sbin/agetty --autologin root -o '-p -- \\u' --keep-baud 115200,38400,9600 %I $TERM
EOF

cat <<EOF > /etc/network/interfaces.d/eth0
auto eth0
allow-hotplug eth0
iface eth0 inet static
address 172.16.0.2/24
gateway 172.16.0.1
EOF

cat <<EOF > /etc/systemd/system/init-entropy.service
[Unit]
Description=Init entropy
Wants=network-online.target
After=network-online.target
[Service]
Type=simple
User=root
ExecStart=python3 /app/entropy.py
[Install]
WantedBy=multi-user.target
EOF
chmod 644 /etc/systemd/system/init-entropy.service
systemctl enable init-entropy.service

cat <<EOF > /etc/systemd/system/function-daemon.service
[Unit]
Description=Serverless function daemon
Wants=init-entropy.service
After=init-entropy.service
StartLimitIntervalSec=0
[Service]
Type=simple
Restart=always
RestartSec=1
User=root
Environment="FLASK_APP=/app/daemon.py"
ExecStart=python3 -m flask run --host=172.16.0.2
[Install]
WantedBy=multi-user.target
EOF

cat <<EOF >> /etc/sysctl.conf
net.ipv6.conf.all.disable_ipv6 = 1
net.ipv6.conf.default.disable_ipv6 = 1
net.ipv6.conf.lo.disable_ipv6 = 1
EOF

# cat <<EOF >> /etc/rsyslog.conf
# *.*    -/dev/shm/syslog
# EOF

cat <<EOF >> /etc/ssh/sshd_config
PermitRootLogin yes
EOF

ln -s /dev/shm /usr/tmp

chmod 644 /etc/systemd/system/function-daemon.service
systemctl enable function-daemon.service

systemctl disable systemd-timesyncd.service
systemctl disable systemd-update-utmp.service
systemctl disable redis-server.service
