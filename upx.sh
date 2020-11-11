#!/bin/sh

# Install UPX
wget https://github.com/upx/upx/releases/download/v3.96/upx-3.96-amd64_linux.tar.xz
tar -xf upx-3.96-amd64_linux.tar.xz

# Compress files
./upx ./dist/shipyard_linux_amd64/shipyard
./upx ./dist/shipyard_linux_arm64/shipyard
./upx ./dist/shipyard_linux_armv7/shipyard
./upx ./dist/shipyard_windows_amd64/shipyard
./upx ./dist/shipyard_darwin_amd64/shipyard

rm -rf upx-3.96-amd64_linux
rm -rf ./upx
rm -rf upx-3.96-amd64_linux.tar.xz
