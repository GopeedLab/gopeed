#!/bin/bash

mkdir /Downloads

chown -R ${PUID}:${PGID} /app /Downloads

umask ${UMASK}

if [ "$#" -gt 0 ]; then
    exec su-exec ${PUID}:${PGID} ./gopeed "$@"
else
    exec su-exec ${PUID}:${PGID} ./gopeed
fi