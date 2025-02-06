# atlas-login
Mushroom game Login Server

## Overview

A stateful, multi-tenant, multi-version login service for a mushroom game.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/
- SERVICE_ID - uuid identifying service
- SERVICE_TYPE - login-service
- COMMAND_TOPIC_ACCOUNT_SESSION
- EVENT_TOPIC_ACCOUNT_SESSION_STATUS
- EVENT_TOPIC_ACCOUNT_STATUS
- EVENT_TOPIC_SESSION_STATUS
