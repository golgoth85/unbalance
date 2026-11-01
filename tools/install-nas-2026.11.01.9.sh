#!/bin/bash
# Da eseguire come root sul NAS Unraid. Non copia configurazioni o dati utente.
set -euo pipefail
umask 077
version=2026.11.01.9
revision=cf6321e3b775b73c1bd106c2f41ee90bb015e43f
plg_sha=a98ac1e77bdaa62adc8de91730226d621b1b34b6143fba68707e67a7049d1d38
bundle_sha=50e5e03d682f13317e05b8e0215143055437c748216d33123df17a77ace80a5e
base="https://github.com/golgoth85/unbalance/releases/download/$version"
code=/usr/local/emhttp/plugins/unbalanced
persistent=/boot/config/plugins/unbalanced
boot_manifest=/boot/config/plugins/unbalanced.plg
registration=/var/log/plugins/unbalanced.plg
runtime=/usr/local/lib/unbalanced-runtime
manager=/usr/local/sbin/plugin
backup= stage= original_hash= original_version=
attempted=0
was_running=0
had_boot=0
had_registration=0
fail() { echo "ERRORE: $*" >&2; exit 1; }
[ "$#" -eq 0 ] || fail "Questo script non accetta argomenti"
[ "$(id -u)" -eq 0 ] && [ -f /etc/unraid-version ] || fail "Eseguire come root sul NAS Unraid"
for cmd in curl sha256sum tar php flock pidof pgrep ps awk sed sort df du cmp; do
    command -v "$cmd" >/dev/null || fail "Comando necessario assente: $cmd"
done
[ -x "$manager" ] && [ -x "$code/scripts/start" ] && [ -x "$code/scripts/stop" ] || fail "Installazione Unbalanced precedente non trovata"
[ -f "$persistent/unbalanced.cfg" ] && [ -f "$persistent/unbalanced.env" ] || fail "Configurazione esistente non trovata"
[ -f "$code/VERSION" ] && [ -f "$code/unbalanced" ] || fail "Codice precedente incompleto"
exec 9>/var/lock/unbalanced-manual-install.lock
flock -n 9 || fail "Un'altra installazione manuale è in corso"
# Leggere soltanto le assegnazioni PORT, senza source e senza stamparle.
port_lines=$(sed -nE '/^[[:space:]]*(export[[:space:]]+)?PORT[[:space:]]*=/p' "$persistent/unbalanced.cfg")
port=7090
if [ -n "$port_lines" ]; then
    [ "$(printf '%s\n' "$port_lines" | wc -l)" -eq 1 ] || fail "PORT duplicata"
    case "$port_lines" in *$'\r'*) fail "PORT con CRLF";; esac
    port=$(printf '%s\n' "$port_lines" | sed -nE \
        -e 's/^[[:space:]]*(export[[:space:]]+)?PORT="([0-9]+)"([[:space:]]+#.*|[[:space:]]*)$/\2/p' \
        -e "s/^[[:space:]]*(export[[:space:]]+)?PORT='([0-9]+)'([[:space:]]+#.*|[[:space:]]*)$/\\2/p" \
        -e 's/^[[:space:]]*(export[[:space:]]+)?PORT=([0-9]+)([[:space:]]+#.*|[[:space:]]*)$/\2/p')
fi
case "$port" in *[!0-9]*|'') fail "PORT non valida";; esac
[ "$port" -gt 0 ] && [ "$port" -le 65535 ] || fail "PORT fuori intervallo"
endpoint="http://127.0.0.1:$port"
read_status() {
    local body
    body=$(curl -fsS --max-time 5 "$endpoint/api/health" 2>/dev/null) || body=""
    if ! printf '%s' "$body" | php -n -r '$s=json_decode(stream_get_contents(STDIN),true); if(!is_array($s)||!isset($s["status"])||!is_int($s["status"]))exit(1); echo $s["status"];' 2>/dev/null; then
        curl -fsS --max-time 5 "$endpoint/api/state" 2>/dev/null | php -n -r '$s=json_decode(stream_get_contents(STDIN),true); if(!is_array($s)||!isset($s["status"])||!is_int($s["status"]))exit(1); echo $s["status"];' 2>/dev/null
    fi
}
assert_idle() {
    local services transfer current parent depth status
    services=$(pidof unbalanced) || services=""
    [ -n "$services" ] || return 0
    for transfer in $(pgrep -x rsync || true); do
        current=$transfer; depth=0
        while [ "$current" -gt 1 ] && [ "$depth" -lt 64 ]; do
            case " $services " in *" $current "*) echo "Rsync di Unbalanced attivo: nessun arresto" >&2; return 1;; esac
            parent=$(ps -o ppid= -p "$current" | tr -d '[:space:]')
            case "$parent" in *[!0-9]*|'')
                [ ! -d "/proc/$current" ] || return 1
                break;;
            esac
            current=$parent; depth=$((depth+1))
        done
        [ "$depth" -lt 64 ] || return 1
    done
    status=$(read_status) || {
        echo "Stato non verificabile sulla porta $port: nessun arresto. Con legacy autenticato, concludere le operazioni e fermare il plugin dal gestore prima di riprovare." >&2
        return 1
    }
    [ "$status" = 0 ] || { echo "Unbalanced occupato: nessun arresto" >&2; return 1; }
}
assert_unchanged() {
    local running_now=0
    if pidof unbalanced >/dev/null; then running_now=1; fi
    [ "$running_now" -eq "$was_running" ] &&
        [ "$(cat "$code/VERSION")" = "$original_version" ] &&
        [ "$(sha256sum "$code/unbalanced" | awk '{print $1}')" = "$original_hash" ] || {
            echo "Codice o stato del servizio cambiato: installazione sospesa" >&2; return 1;
        }
}
healthy() {
    curl -fsS --max-time 3 "$endpoint/api/health" 2>/dev/null | php -n -r '$s=json_decode(stream_get_contents(STDIN),true); exit(is_array($s)&&($s["version"]??null)===$argv[1]&&($s["status"]??null)===0?0:1);' "$version" 2>/dev/null
}
restore_entry() {
    local saved=$1 target=$2 present=$3
    [ ! -d "$target" ] || return 1
    if [ -e "$target" ] || [ -L "$target" ]; then unlink "$target" || return 1; fi
    if [ "$present" -eq 1 ]; then cp -a "$saved" "$target" || return 1; fi
}
rollback() {
    local current_hash current_version status restored=0
    current_hash=$(sha256sum "$code/unbalanced" 2>/dev/null | awk '{print $1}') || current_hash=""
    current_version=$(cat "$code/VERSION" 2>/dev/null) || current_version=""
    # L'installer della release ripristina già il codice sui suoi errori.
    if [ "$current_hash" != "$original_hash" ] || [ "$current_version" != "$original_version" ]; then
        assert_idle || { echo "Rollback automatico sospeso per non interrompere attività; backup: $backup" >&2; return 1; }
        if pidof unbalanced >/dev/null; then "$code/scripts/stop" || return 1; fi
        mkdir -p "$stage/restore" || return 1
        tar -xzf "$backup/code.tgz" -C "$stage/restore" || return 1
        chmod -R a+rX "$stage/restore/unbalanced" || return 1
        if [ -d "$code" ]; then mv "$code" "$stage/failed-code" || return 1; fi
        mv "$stage/restore/unbalanced" "$code" || return 1
    fi
    restore_entry "$backup/boot-manifest.plg" "$boot_manifest" "$had_boot" || return 1
    restore_entry "$backup/registration" "$registration" "$had_registration" || return 1
    if [ "$was_running" -eq 1 ]; then
        if ! pidof unbalanced >/dev/null; then "$code/scripts/start" 9>&- || return 1; fi
        for attempt in 1 2 3 4 5 6 7 8 9 10; do
            status=$(read_status) || status=""
            if [ "$status" = 0 ]; then restored=1; break; fi
            sleep 1
        done
        [ "$restored" -eq 1 ] || { echo "Codice ripristinato ma stato del servizio precedente non confermato" >&2; return 1; }
    elif pidof unbalanced >/dev/null; then
        assert_idle || return 1
        "$code/scripts/stop" || return 1
    fi
    echo "Ripristino codice, manifest e stato precedente completato; backup: $backup"
}
finish() {
    local result=$?
    trap - EXIT
    if [ "$result" -ne 0 ]; then
        if [ "$attempted" -eq 1 ]; then rollback || echo "Controllo manuale necessario; codice/manifest salvati in $backup" >&2; fi
        [ -z "$stage" ] || echo "Materiale diagnostico conservato in $stage" >&2
    elif [ -n "$stage" ]; then
        find "$stage" -depth -delete
    fi
    exit "$result"
}
trap finish EXIT
assert_idle || fail "Installazione sospesa senza modificare il servizio"
if pidof unbalanced >/dev/null; then was_running=1; fi
original_version=$(cat "$code/VERSION")
original_hash=$(sha256sum "$code/unbalanced" | awk '{print $1}')
if [ "$original_version" = "$version" ] && [ -f "$code/RELEASE_REVISION" ] && [ "$(cat "$code/RELEASE_REVISION")" = "$revision" ]; then
    if [ "$was_running" -eq 1 ]; then healthy || fail "Revisione già presente ma salute non confermata"; fi
    echo "Revisione $revision già installata: nessun backup o riavvio aggiuntivo"
    exit 0
fi
case "$original_version" in *[!0-9.a-zA-Z-]*|'') fail "Versione precedente non valida";; esac
[ "$(printf '%s\n%s\n' "$original_version" "$version" | sort -V | tail -n 1)" = "$version" ] || fail "Non eseguo downgrade di una versione più recente"
mkdir -p "$runtime"; chmod 700 "$runtime"
stage=$(mktemp -d "$runtime/manual.XXXXXX")
for name in unbalanced.plg "unbalanced-$version.tgz"; do
    curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' --retry 2 --connect-timeout 10 --max-time 180 "$base/$name" -o "$stage/$name"
done
printf '%s  %s\n' "$plg_sha" "$stage/unbalanced.plg" "$bundle_sha" "$stage/unbalanced-$version.tgz" | sha256sum -c -
# Backup persistente separato dalla retention automatica dell'installer.
free_kb=$(df -Pk "$persistent" | awk 'NR==2 {print $4}')
needed_kb=$(( $(du -sk "$code" | awk '{print $1}') + $(du -k "$stage/unbalanced-$version.tgz" | awk '{print $1}') + 16384 ))
[ "$free_kb" -ge "$needed_kb" ] || fail "Spazio insufficiente sulla chiavetta per backup e pacchetto"
assert_idle || fail "Attività iniziata durante il download: installazione sospesa"
assert_unchanged || fail "Stato precedente non più valido"
backup=$(mktemp -d "$persistent/manual-install-$version-$(date -u +%Y%m%dT%H%M%SZ).XXXXXX")
tar --exclude='*.env' --exclude='*.cfg' --exclude='secrets' --exclude='gateway-secrets' -czf "$backup/code.tgz" -C /usr/local/emhttp/plugins unbalanced
if [ -e "$boot_manifest" ] || [ -L "$boot_manifest" ]; then cp -a "$boot_manifest" "$backup/boot-manifest.plg"; had_boot=1; fi
if [ -e "$registration" ] || [ -L "$registration" ]; then cp -a "$registration" "$backup/registration"; had_registration=1; fi
if ! printf '%s  %s\n' "$bundle_sha" "$persistent/unbalanced-$version.tgz" | sha256sum -c - >/dev/null 2>&1; then
    cp "$stage/unbalanced-$version.tgz" "$persistent/.unbalanced-$version.download"
    mv "$persistent/.unbalanced-$version.download" "$persistent/unbalanced-$version.tgz"
fi
assert_idle || fail "Installazione sospesa: nuova attività"
assert_unchanged || fail "Stato precedente non più valido"
echo "Installo $version; backup codice e manifest: $backup"
attempted=1
"$manager" install "$stage/unbalanced.plg" 9>&- > "$backup/install.log" 2>&1 || fail "Gestore plugin terminato con errore; log locale: $backup/install.log"
[ "$(cat "$code/VERSION")" = "$version" ] && [ "$(cat "$code/RELEASE_REVISION")" = "$revision" ] || fail "Revisione installata inattesa"
cmp -s "$stage/unbalanced.plg" "$boot_manifest" || fail "Manifest persistente non corrispondente alla release"
if [ "$was_running" -eq 1 ]; then
    ok=0
    for attempt in 1 2 3 4 5 6 7 8 9 10; do
        if healthy; then ok=1; break; fi
        sleep 1
    done
    [ "$ok" -eq 1 ] || fail "Salute della nuova versione non confermata"
    echo "Installazione completata: $version, revisione $revision, salute HTTP confermata sulla porta $port"
else
    echo "Installazione completata: $version. Servizio conservato fermo; salute HTTP non verificata."
fi
attempted=0
echo "Configurazione e cronologia non modificate. Backup conservato: $backup"
