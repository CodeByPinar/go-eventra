# Eventra Auth API (v1)

## Health

- Method: `GET`
- URL: `/health`

## Register

- Method: `POST`
- URL: `/api/v1/auth/register`

Request:

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "strongpassword123"
}
```

Response (201):

```json
{
  "token": "<jwt>",
  "access_token": "<jwt>",
  "refresh_token": "<refresh-token>",
  "user": {
    "id": "9ff2163d-a8d0-45a5-b66d-3e7f0c7dcbbf",
    "username": "alice",
    "email": "alice@example.com",
    "created_at": "2026-03-17T13:05:40Z"
  }
}
```

## Login

- Method: `POST`
- URL: `/api/v1/auth/login`

Request:

```json
{
  "email": "alice@example.com",
  "password": "strongpassword123"
}
```

Response (200):

```json
{
  "token": "<jwt>",
  "access_token": "<jwt>",
  "refresh_token": "<refresh-token>",
  "user": {
    "id": "9ff2163d-a8d0-45a5-b66d-3e7f0c7dcbbf",
    "username": "alice",
    "email": "alice@example.com",
    "created_at": "2026-03-17T13:05:40Z"
  }
}
```

## Refresh

- Method: `POST`
- URL: `/api/v1/auth/refresh`

Request:

```json
{
  "refresh_token": "<refresh-token>"
}
```

Response (200):

```json
{
  "token": "<jwt>",
  "access_token": "<jwt>",
  "refresh_token": "<new-refresh-token>",
  "user": {
    "id": "9ff2163d-a8d0-45a5-b66d-3e7f0c7dcbbf",
    "username": "alice",
    "email": "alice@example.com",
    "created_at": "2026-03-17T13:05:40Z"
  }
}
```

## Logout

- Method: `POST`
- URL: `/api/v1/auth/logout`

Request:

```json
{
  "refresh_token": "<refresh-token>"
}
```

Response (204): empty body

## Me (Protected)

- Method: `GET`
- URL: `/api/v1/auth/me`
- Header: `Authorization: Bearer <jwt>`

Response (200):

```json
{
  "user_id": "9ff2163d-a8d0-45a5-b66d-3e7f0c7dcbbf",
  "email": "alice@example.com"
}
```
