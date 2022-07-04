#!/bin/sh

/lcpserver/entrypoint.sh /lcpserver/tmp/config.yaml /lcpserver/lcpserver &
echo "LCPSERVER PID=$!"


/lsdserver/entrypoint.sh /lsdserver/tmp/config.yaml /lsdserver/lsdserver &
echo "LSDSERVER PID=$!"


/frontend/entrypoint.sh /frontend/tmp/config.yaml /frontend/manage /frontend/frontend &
echo "FRONTENDSERVER PID=$!"


