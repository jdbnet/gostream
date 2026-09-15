#!/bin/sh
set -e

if [ "$(id -u)" = "0" ]; then
	mkdir -p /home/gostream/.config/gostream \
		/home/gostream/.local/share/gostream/media/tracks \
		/home/gostream/.local/share/gostream/media/jingles \
		/home/gostream/.local/share/gostream/media/artworks
	chown -R gostream:gostream /home/gostream/.config /home/gostream/.local
	cd /app
	exec gosu gostream "$@"
fi

exec "$@"
