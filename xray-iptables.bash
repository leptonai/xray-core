#!/bin/bash

# Install/Remove XRAY Transparent Proxy IPTables on client-side machine

remove() {
  iptables -t nat -F XRAY || true
  # Now only we add the iptable chain named XRAY, so `iptables -t nat -L PREROUTING --line-numbers | grep XRAY | awk '{print $1}'` returns only one result.
  iptables -t nat -D PREROUTING `iptables -t nat -L PREROUTING --line-numbers | grep XRAY | awk '{print $1}'` || true
  iptables -t nat -D OUTPUT `iptables -t nat -L OUTPUT --line-numbers | grep XRAY | awk '{print $1}'` || true
  iptables -t nat -X XRAY || true
}

if [[ "$#" -gt '0' ]]; then
  case "$1" in
    '--remove')
      remove
      echo "XRAY iptables removed"
      exit 0
      ;;
  esac
fi

# 100.64.0.0/10 is tailscale ip range
PrivateIPRanges="100.64.0.0/10,10.0.0.0/8,127.0.0.0/8,0.0.0.0/8,169.254.0.0/16,172.16.0.0/12,192.168.0.0/16,224.0.0.0/4,240.0.0.0/4"

iptables -t nat -N XRAY || echo "The XRAY chain already exists"

# Ignore SO_MARK with 0xff
iptables -t nat -C XRAY -p tcp -j RETURN -m mark --mark 0xff -m comment --comment "Ignore traffic marked 0xff" || iptables -t nat -A XRAY -p tcp -j RETURN -m mark --mark 0xff -m comment --comment "Ignore traffic marked 0xff"

# Ignore LANs and any other addresses you'd like to bypass the proxy
# See Wikipedia and RFC5735 for full list of reserved networks.
iptables -t nat -C XRAY -d ${PrivateIPRanges} -j RETURN -m comment --comment "Ignore traffic to private ip"  || iptables -t nat -A XRAY -d ${PrivateIPRanges} -j RETURN -m comment --comment "Ignore traffic to private ip"

# Traffic from the LOCAL to destport 80/8080/443 should DNAT to the Dokodemo-door
iptables -t nat -C XRAY -p tcp -m addrtype --src-type LOCAL -j DNAT --dport 80   --to-destination 127.0.0.1:12345 -m comment --comment "Lepton managed rule"  || iptables -t nat -A XRAY -p tcp -m addrtype --src-type LOCAL -j DNAT --dport 80   --to-destination 127.0.0.1:12345 -m comment --comment "Lepton managed rule"
iptables -t nat -C XRAY -p tcp -m addrtype --src-type LOCAL -j DNAT --dport 8080 --to-destination 127.0.0.1:12345 -m comment --comment "Lepton managed rule"  || iptables -t nat -A XRAY -p tcp -m addrtype --src-type LOCAL -j DNAT --dport 8080 --to-destination 127.0.0.1:12345 -m comment --comment "Lepton managed rule"
iptables -t nat -C XRAY -p tcp -m addrtype --src-type LOCAL -j DNAT --dport 443  --to-destination 127.0.0.1:12345 -m comment --comment "Lepton managed rule"  || iptables -t nat -A XRAY -p tcp -m addrtype --src-type LOCAL -j DNAT --dport 443  --to-destination 127.0.0.1:12345 -m comment --comment "Lepton managed rule"
iptables -t nat -C XRAY -p tcp -s ${PrivateIPRanges} -j DNAT --dport 80   --to-destination 127.0.0.1:12345 -m comment --comment "Lepton managed rule"         || iptables -t nat -A XRAY -p tcp -s ${PrivateIPRanges} -j DNAT --dport 80   --to-destination 127.0.0.1:12345 -m comment --comment "Lepton managed rule"
iptables -t nat -C XRAY -p tcp -s ${PrivateIPRanges} -j DNAT --dport 8080 --to-destination 127.0.0.1:12345 -m comment --comment "Lepton managed rule"         || iptables -t nat -A XRAY -p tcp -s ${PrivateIPRanges} -j DNAT --dport 8080 --to-destination 127.0.0.1:12345 -m comment --comment "Lepton managed rule"
iptables -t nat -C XRAY -p tcp -s ${PrivateIPRanges} -j DNAT --dport 443  --to-destination 127.0.0.1:12345 -m comment --comment "Lepton managed rule"         || iptables -t nat -A XRAY -p tcp -s ${PrivateIPRanges} -j DNAT --dport 443  --to-destination 127.0.0.1:12345 -m comment --comment "Lepton managed rule"

# Apply the rules
iptables -t nat -C PREROUTING -p tcp -j XRAY -m comment --comment "add XRAY into PREROUTING" || iptables -t nat -A PREROUTING -p tcp -j XRAY -m comment --comment "add XRAY into PREROUTING"
iptables -t nat -C OUTPUT     -p tcp -j XRAY -m comment --comment "add XRAY into OUTPUT"     || iptables -t nat -A OUTPUT     -p tcp -j XRAY -m comment --comment "add XRAY into OUTPUT"
