#!/bin/bash

nssh run -A -J lb.bastion.nemax.nebiuscloud.net --ycp-profile nemax --no-yubikey  \
  "(echo 'hello world')" $HOSTNAME
