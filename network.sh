#!/usr/bin/env bash

set -eux

# network

idx=${1}

ip netns add fc$idx
ip netns exec fc$idx ip tuntap add name vmtap0 mode tap
ip netns exec fc$idx ip addr add 172.16.0.1/24 dev vmtap0
ip netns exec fc$idx ip link set vmtap0 up
# create the veth pair inside the namespace
ip netns exec fc$idx ip link add veth$idx type veth peer name veth0
# move vethidx to the global host namespace
ip netns exec fc$idx ip link set veth$idx netns 1
veth0_ip=$(printf 10.1.%d.2 ${idx})
vethidx_ip=$(printf 10.1.%d.1 ${idx})
ip netns exec fc$idx ip addr add ${veth0_ip}/24 dev veth0
ip netns exec fc$idx ip link set dev veth0 up
ip addr add ${vethidx_ip}/24 dev veth$idx
ip link set dev veth$idx up
# designate the outer end as default gateway for packets leaving the namespace
sudo ip netns exec fc$idx ip route add default via $vethidx_ip
# for packets that leave the namespace and have the source IP address of the
# original guest, rewrite the source address to clone address 192.168.0.3
sudo ip netns exec fc$idx iptables -t nat -A POSTROUTING -o veth0 \
-s 172.16.0.2 -j SNAT --to 192.168.0.$((idx+2))
# do the reverse operation; rewrites the destination address of packets
# heading towards the clone address to 192.168.241.2
sudo ip netns exec fc$idx iptables -t nat -A PREROUTING -i veth0 \
-d 192.168.0.$((idx+2)) -j DNAT --to-destination 172.16.0.2

# (adds a route on the host for the clone address)
sudo ip route add 192.168.0.$((idx+2)) via $veth0_ip
