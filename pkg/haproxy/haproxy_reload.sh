#!/usr/bin/env bash
set -euo pipefail

pid_file=/var/run/haproxy.pid
candidate_path=/etc/haproxy/haproxy_new.cfg

validate_config() {
    haproxy -c -f "$candidate_path"
}

running_pids() {
    [[ -s "$pid_file" ]] || return 1
    local pid
    while read -r pid; do
        [[ "$pid" =~ ^[0-9]+$ ]] || return 1
        kill -0 "$pid" 2>/dev/null || return 1
    done < "$pid_file"
}

start_haproxy() {
    local config_path="$1"
    rm -f "$pid_file"
    haproxy -p "$pid_file" -f "$config_path"
}

reload_haproxy() {
    local config_path="$1"
    if ! running_pids; then
        start_haproxy "$config_path"
        return
    fi

    local old_pids=()
    mapfile -t old_pids < "$pid_file"
    haproxy -p "$pid_file" -f "$config_path" -sf "${old_pids[@]}"
}

apply_config() {
    local config_path="$1"
    local action="$2"
    if [[ "$action" == "start" ]]; then
        echo "validating and starting haproxy"
        validate_config
        install -m 0644 "$candidate_path" "$config_path"
        reload_haproxy "$config_path"
    elif ! cmp -s "$config_path" "$candidate_path"; then
        echo "reloading haproxy configuration"
        validate_config
        install -m 0644 "$candidate_path" "$config_path"
        reload_haproxy "$config_path"
    else
        echo "haproxy configuration unchanged"
    fi
}

if [[ "$#" -ne 2 ]] || { [[ "$2" != "start" ]] && [[ "$2" != "reload" ]]; }; then
    echo "usage: $0 <config-path> <start|reload>" >&2
    exit 2
fi

apply_config "$1" "$2"
