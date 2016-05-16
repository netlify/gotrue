# Authlify - User mangament for APIs

Authlify is a small open-source API that can act as a self-standing API service
for handling user registration and authentication for JAM projects.

It's based on OAuth2 and JWT and will handle user signup and authentication.

It exposes the following endpoints:

* POST /signup

  ```json
  {
    "email": "email@example.com",
    "password": "secret"
  }
  ```

  Returns:

  ```json
  {
    "id":1,
    "email":"email@example.com",
    "confirmation_sent_at": "2016-05-15T20:49:40.882805774-07:00",
    "created_at": "2016-05-15T19:53:12.368652374-07:00",
    "updated_at": "2016-05-15T19:53:12.368652374-07:00"
  }
  ```

* POST /verify

  ```json
  {
    "token": "confirmation-code-delivered-in-email"
  }
  ```

  Returns:

  ```json
  {
    "id":1,
    "email":"email@example.com",
    "confirmation_sent_at": "2016-05-15T20:49:40.882805774-07:00",
    "created_at": "2016-05-15T19:53:12.368652374-07:00",
    "updated_at": "2016-05-15T19:53:12.368652374-07:00"
  }
  ```

* POST /token

  This is an OAuth2 endpoint that currently implements
  the password and refresh_token grant types

  ```
  grant_type=password&username=email@example.com&password=secret
  ```

  or

  ```
  grant_type=refresh_token&refresh_token=my-refresh-token
  ```

  Returns:

  ```
  {
    "access_token": "jwt-token-representing-the-user",
    "token_type": "bearer",
    "expires_in": 3600,
    "refresh_token": "a-refresh-token"
  }
  ```
