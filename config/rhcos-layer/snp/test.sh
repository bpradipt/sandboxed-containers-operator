#!/bin/bash

trustee_url=${trustee_url:-http://test}
text="[hypervisor.qemu]
kernel_params= \"agent.aa_kbc_params=cc_kbc::$trustee_url\""
encoded_text=$(echo "$text" | base64 -w0)
echo $encoded_text
