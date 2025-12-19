# atlas-account

Mushroom game Account Service

## Overview

A RESTful resource which provides account services.

## Environment

### General
- JAEGER_HOST_PORT - Jaeger [host]:[port] for distributed tracing
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace

### Database
- DB_USER - Postgres user name
- DB_PASSWORD - Postgres user password
- DB_HOST - Postgres Database host
- DB_PORT - Postgres Database port
- DB_NAME - Postgres Database name

### Kafka
- BOOTSTRAP_SERVERS - Kafka [host]:[port]

#### Kafka Topics
- EVENT_TOPIC_ACCOUNT_STATUS - Kafka Topic for transmitting Account Status Events (CREATED, LOGGED_IN, LOGGED_OUT)
- EVENT_TOPIC_ACCOUNT_SESSION_STATUS - Kafka Topic for transmitting Account Session Status Events (CREATED, STATE_CHANGED, REQUEST_LICENSE_AGREEMENT, ERROR)
- COMMAND_TOPIC_CREATE_ACCOUNT - Kafka Topic for receiving Create Account Commands
- COMMAND_TOPIC_ACCOUNT_SESSION - Kafka Topic for receiving Account Session Commands (CREATE, PROGRESS_STATE, LOGOUT)

## API

All API endpoints are prefixed with `/api/`.

### Headers

All RESTful requests require the following header information to identify the server instance:

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Endpoints

#### Get All Accounts

- **URL**: `/api/accounts/`
- **Method**: `GET`
- **Description**: Retrieves all accounts for the current tenant.
- **Response**: Array of Account objects
- **Response Format**:
  ```json
  [
    {
      "name": "accountName",
      "pin": "1234",
      "pic": "123456",
      "loggedIn": 0,
      "lastLogin": 0,
      "gender": 0,
      "banned": false,
      "tos": true,
      "language": "en",
      "country": "us",
      "characterSlots": 4
    }
  ]
  ```
- **Status Codes**:
  - `200 OK`: Successfully retrieved accounts
  - `500 Internal Server Error`: Server error

#### Get Account By ID

- **URL**: `/api/accounts/{accountId}`
- **Method**: `GET`
- **URL Parameters**: 
  - `accountId` - The ID of the account to retrieve
- **Description**: Retrieves a specific account by its ID.
- **Response**: Account object
- **Response Format**:
  ```json
  {
    "name": "accountName",
    "pin": "1234",
    "pic": "123456",
    "loggedIn": 0,
    "lastLogin": 0,
    "gender": 0,
    "banned": false,
    "tos": true,
    "language": "en",
    "country": "us",
    "characterSlots": 4
  }
  ```
- **Status Codes**:
  - `200 OK`: Successfully retrieved account
  - `404 Not Found`: Account not found
  - `400 Bad Request`: Invalid account ID

#### Get Account By Name

- **URL**: `/api/accounts?name={name}`
- **Method**: `GET`
- **Query Parameters**: 
  - `name` - The name of the account to retrieve
- **Description**: Retrieves a specific account by its name.
- **Response**: Account object
- **Response Format**:
  ```json
  {
    "name": "accountName",
    "pin": "1234",
    "pic": "123456",
    "loggedIn": 0,
    "lastLogin": 0,
    "gender": 0,
    "banned": false,
    "tos": true,
    "language": "en",
    "country": "us",
    "characterSlots": 4
  }
  ```
- **Status Codes**:
  - `200 OK`: Successfully retrieved account
  - `404 Not Found`: Account not found
  - `400 Bad Request`: Missing name parameter

#### Create Account

- **URL**: `/api/accounts/`
- **Method**: `POST`
- **Description**: Creates a new account.
- **Request Body**:
  ```json
  {
    "name": "accountName",
    "password": "password123",
    "gender": 0
  }
  ```
- **Status Codes**:
  - `202 Accepted`: Account creation request accepted
  - `400 Bad Request`: Invalid request body

#### Update Account

- **URL**: `/api/accounts/{accountId}`
- **Method**: `PATCH`
- **URL Parameters**: 
  - `accountId` - The ID of the account to update
- **Description**: Updates an existing account.
- **Request Body**: Account object with fields to update
- **Response**: Updated Account object
- **Status Codes**:
  - `200 OK`: Successfully updated account
  - `404 Not Found`: Account not found
  - `400 Bad Request`: Invalid request body or account ID

#### Delete Account Session

- **URL**: `/api/accounts/{accountId}/session`
- **Method**: `DELETE`
- **URL Parameters**: 
  - `accountId` - The ID of the account whose session to delete
- **Description**: Logs out an account by deleting its session.
- **Status Codes**:
  - `202 Accepted`: Logout request accepted
  - `400 Bad Request`: Invalid account ID
