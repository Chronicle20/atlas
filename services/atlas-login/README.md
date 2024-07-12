# atlas-login
Mushroom game Login Server

## Overview

A stateful, multi-tenant, multi-version login service for a mushroom game.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- CONFIG_FILE - Location of service configuration file.
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- COMMAND_TOPIC_ACCOUNT_LOGOUT - Kafka Topic for transmitting Account Logout Commands
- ACCOUNT_SERVICE_URL - [scheme]://[host]:[port]/api/aos/
- CHARACTER_SERVICE_URL - [scheme]://[host]:[port]/api/cos/
- CHARACTER_FACTORY_SERVICE_URL - [scheme]://[host]:[port]/api/cfs/
- WORLD_SERVICE_URL - [scheme]://[host]:[port]/api/wrg/
