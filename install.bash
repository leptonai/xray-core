#!/bin/bash

# Install/Remove XRAY

BINPath="/usr/local/bin"
XRayPath="/usr/local/etc/xray"
LogPath="/var/log/xray"
ServicePath="/etc/systemd/system"


install(){
  mkdir -p ${XRayPath}
  mkdir -p ${LogPath}
  
  cp xray ${BINPath}
  cp xray-manager ${BINPath}
  cp -r conf.d ${XRayPath}
  cp geoip.dat ${XRayPath}
  cp geosite.dat ${XRayPath}
  cp xray-iptables.bash ${XRayPath}
  cp xray.service ${ServicePath}
  cp xray-manager.service ${ServicePath}

  systemctl daemon-reload
  systemctl enable --now xray.service
  systemctl enable --now xray-manager.service
}

removeV2RAYIPTables() {
  iptables -t nat -F V2RAY || true
  # Now only we add the iptable chain named V2RAY, so `iptables -t nat -L PREROUTING --line-numbers | grep V2RAY | awk '{print $1}'` returns only one result.
  iptables -t nat -D PREROUTING `iptables -t nat -L PREROUTING --line-numbers | grep V2RAY | awk '{print $1}'` || true
  iptables -t nat -D OUTPUT `iptables -t nat -L OUTPUT --line-numbers | grep V2RAY | awk '{print $1}'` || true
  iptables -t nat -X V2RAY || true
  systemctl disable --now v2ray.service || true
}

remove() {
  systemctl disable --now xray-manager.service || true
  ${XRayPath}/xray-iptables.bash --remove || true
  systemctl disable --now xray.service || true
}


if [[ "$#" -gt '0' ]]; then
  case "$1" in
    '--remove')
      remove
      echo "v2ray iptables removed"
      exit 0
      ;;
  esac
fi

removeV2RAYIPTables
install
echo "install xray components successfully"
