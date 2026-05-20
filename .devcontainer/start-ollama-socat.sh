#!/bin/sh

set -eu

pidfile=/tmp/ollama-socat.pid
logfile=/tmp/ollama-socat.log
upstream="${OLLAMA_UPSTREAM:-host.docker.internal:11434}"

if ss -ltnH '( sport = :11434 )' 2>/dev/null | grep -Fq '127.0.0.1:11434'; then
	exit 0
fi

if [ -f "$pidfile" ]; then
	pid="$(cat "$pidfile" 2>/dev/null || true)"
	if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
		command_name="$(ps -p "$pid" -o comm= 2>/dev/null | tr -d '[:space:]')"
		if [ "$command_name" = "socat" ]; then
			exit 0
		fi
	fi
	rm -f "$pidfile"
fi

nohup setsid socat TCP-LISTEN:11434,bind=127.0.0.1,fork,reuseaddr "TCP:${upstream}" \
	>"$logfile" 2>&1 < /dev/null &
echo $! > "$pidfile"