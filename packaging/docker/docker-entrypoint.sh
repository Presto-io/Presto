#!/bin/sh
set -eu

if [ "${1#-}" != "$1" ]; then
	set -- presto-server "$@"
fi

config_dir="${PRESTO_CONFIG_DIR:-/config}"
data_dir="${PRESTO_DATA_DIR:-/data}"
cache_dir="${PRESTO_CACHE_DIR:-/cache}"
log_dir="${PRESTO_LOG_DIR:-/logs}"

if [ "$(id -u)" = "0" ]; then
	mkdir -p "$config_dir" "$data_dir/templates" "$data_dir/fonts" "$data_dir/runtimes" "$cache_dir" "$log_dir"
	chown presto:presto "$data_dir"
	chown -R presto:presto "$config_dir" "$data_dir/templates" "$data_dir/runtimes" "$cache_dir" "$log_dir"
	exec su-exec presto "$@"
fi

exec "$@"
