#!/bin/bash

chown -R ${PUID}:${PGID} /app/Download

umask ${UMASK}

exec su-exec ${PUID}:${PGID} ./gopeed "$@"