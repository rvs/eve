#!/bin/sh

mkdir -p /var/lib/tailscale
cp /config/tailscale.conf /var/lib/tailscale/relay.conf

mkdir -p /persist/tailscale/cache
export CACHE_DIRECTORY=/persist/tailscale/cache

# start tailscale daemon
/bin/tailscaled --state=/persist/tailscale/tailscaled.state &

# start debug ssh server
/bin/tsshd --hostkey=/etc/ssh/ssh_host_rsa_key &

while true ; do
  sleep 60
done
