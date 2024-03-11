#!/bin/bash

nssh run -A -J lb.bastion.nemax.nebiuscloud.net --ycp-profile nemax --no-yubikey \
  "(sudo systemctl restart ydb-server-storage.service)" $HOSTNAME
