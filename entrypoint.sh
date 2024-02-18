#!/bin/bash

chown -R ${PUID}:${PGID} /app /Downloads

umask ${UMASK}

exec su-exec ${PUID}:${PGID} ./gopeed "$@"