#!/bin/bash
# Compila e prepara il pacchetto, senza pubblicare o installare sul NAS.
set -euo pipefail
version=${1:?Versione richiesta}
out=${2:?Output richiesto sotto /temp/unbalance}
case "$out" in /temp/unbalance/*) ;; *) echo 'Output non ammesso' >&2; exit 1 ;; esac
case "$version" in *[!0-9.]*|'') echo 'Versione non valida' >&2; exit 1 ;; esac
: "${GOCACHE:?Impostare cache Go sotto /temp/unbalance}"
: "${GOMODCACHE:?Impostare moduli Go sotto /temp/unbalance}"
mkdir -p "$out/package/unbalanced"
(cd ui && npm run build)
cp -a meta/plugin/. "$out/package/unbalanced/"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-X main.Version=$version" -o "$out/package/unbalanced/unbalanced" .
printf '%s\n' "$version" > "$out/package/unbalanced/VERSION"
git rev-parse HEAD > "$out/package/unbalanced/RELEASE_REVISION"
chmod -R a+rX "$out/package"
tar --owner=0 --group=0 -czf "$out/unbalanced-$version.tgz" -C "$out/package" unbalanced
md5=$(md5sum "$out/unbalanced-$version.tgz" | cut -d' ' -f1)
sha256=$(sha256sum "$out/unbalanced-$version.tgz" | cut -d' ' -f1)
sed -e "s/##version##/$version/g" -e "s/##md5##/$md5/g" -e "s/##sha256##/$sha256/g" -e 's/##changes##/- Protezione conflitti GATHER, share divise, filtri rsync, STOP e installer corretti./g' meta/template/unbalanced.plg > "$out/unbalanced.plg"
(cd "$out" && sha256sum "unbalanced-$version.tgz" unbalanced.plg > SHA256SUMS)
echo "Pacchetto pronto: $out"
