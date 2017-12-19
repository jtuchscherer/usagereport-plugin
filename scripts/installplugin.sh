#!/bin/bash

set -e

(cf uninstall-plugin "usage-report" || true) && go build -o usagereport-plugin usagereport.go && cf install-plugin -f usagereport-plugin
