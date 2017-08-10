# GoTrue - User management for APIs

GoTrue is a small open-source API written in golang, that can act as a self-standing
API service for handling user registration and authentication for JAM projects.

It's based on OAuth2 and JWT and will handle user signup, authentication and custom
user data.

## Configuration

You may configure GoTrue using either a configuration file named `config.json`,
environment variables, or a combination of both. To determine the environment variable name:
uppercase the setting name, replace all `.` with `_`, and add a prefix of `GOTRUE_`.

### Top-Level

```
{
  "site_url": "https://example.netlify.com/",
  ...
}
```

`site_url` - `string` **required**

The base URL your site is located at. Currently used in combination with other settings to construct URLs used in emails.

`netlify_secret` - `string` *Multi-instance mode only*

The shared secret with Netlify for this microservice. Used to verify requests have been proxied through Netlify and
the payload values can be trusted.

### API

```
{
  ...
  "api": {
    "host": "localhost",
    "port": 9999
  },
  ...
}
```

`api.host` - `string` **required**

Hostname to listen on.

`api.port` - `number` **required**

Port number to listen on. Can also be set with `PORT` environment variable.

`api.endpoint` - `string` *Multi-instance mode only*

Controls what endpoint Netlify can access this API on.

### Database

```
{
  ...
  "db": {
    "driver": "sqlite3",
    "url": "gotrue.db"
  },
  ...
}
```

`db.driver` - `string` **required**

Chooses what dialect of database you want. Choose from `mongo`, `sqlite3`, `mysql`, or `postgres`.

`db.url` - `string` **required**

Connection string for the database. See the [gorm examples](https://github.com/jinzhu/gorm/blob/gh-pages/documents/database.md) for more details.

`db.namespace` - `string`

Adds a prefix to all table names.

`db.automigrate` - `bool`

If enabled, creates missing tables and columns upon startup.

### Logging

```
{
  ...
  "log_conf": {
    "log_level": "debug"
  },
  ...
}
```

`log_conf.log_level` - `string`

Controls what log levels are output. Choose from `panic`, `fatal`, `error`, `warn`, `info`, or `debug`. Defaults to `info`.

`log_conf.log_file` - `string`

If you wish logs to be written to a file, set `log_file` to a valid file path.

### JSON Web Tokens (JWT)

```
{
  ...
  "jwt": {
    "secret": "supersecretvalue",
    "exp": 3600,
    "aud": "netlify"
  },
  ...
}
```

`jwt.secret` - `string` **required**

The secret used to sign JWT tokens with.

`jwt.exp` - `number`

How long tokens are valid for, in seconds. Defaults to 3600 (1 hour).

`jwt.aud` - `string`

The default JWT audience. Use audiences to group users.

`jwt.admin_group_name` - `string`

The name of the admin group (if enabled). Defaults to `admin`.

`jwt.admin_group_disabled` - `bool`

The first user created will be made an admin.
Set to `true` to turn off this behavior.
Defaults to `false`.

`jwt.default_group_name` - `string`

The default group to assign all new users to.

### External Authentication Providers

We support `github`, `gitlab`, and `bitbucket` for external authentication.
Use the names as the keys underneath `external` to configure each separately.

```
{
  ...
  "external": {
    "github": {
      "client_id": "myappclientid",
      "secret": "clientsecretvaluessssh"
    },
    ...
  },
  ...
}
```

No external providers are required, but you must provide the required values if you choose to enable any.

`external.x.client_id` - `string` **required**

The OAuth2 Client ID registered with the external provider.

`external.x.secret` - `string` **required**

The OAuth2 Client Secret provided by the external provider when you registered.

`external.x.redirect_uri` - `string` **required for gitlab**

The URI a OAuth2 provider will redirect to with the `code` and `state` values.

`external.x.url` - `string`

The base URL used for constructing the URLs to request authorization and access tokens. Used by `gitlab` only. Defaults to `https://gitlab.com`.

### E-Mail

Sending email is not required, but highly recommended for password recovery.
If enabled, you must provide the required values below.

```
{
  ...
  "mailer": {
    "admin_email": "support@example.com",
    "host": "",
    "port": 25,
    "user": "",
    "pass": "",
    "subjects": {
      "confirmation": "",
      ...
    },
    "templates": {
      "confirmation": "",
      ...
    }
  },
  ...
}
```

`mailer.admin_email` - `string` **required**

The `From` email address for all emails sent.

`mailer.host` - `string` **required**

The mail server hostname to send emails through.

`mailer.port` - `number` **required**

The port number to connect to the mail server on.

`mailer.user` - `string`

If the mail server requires authentication, the username to use.

`mailer.pass` - `string`

If the mail server requires authentication, the password to use.

`mailer.max_frequency` - `number`

Controls the minimum amount of time that must pass before sending another signup confirmation or password reset email. The value is the number of seconds. Defaults to 900 (15 minutes).

`mailer.autoconfirm` - `bool`

If you do not require email confirmation, you may set this to `true`. Defaults to `false`.

`mailer.member_folder` - `string`

The folder on `site_url` where the `confirm`, `recover`, and `confirm-email`
pages are located. This is used in combination with `site_url` to generate the 
URLs used in the emails.

`mailer.subjects.confirmation` - `string`

Email subject to use for signup confirmation. Defaults to `Confirm Your Signup`.

`mailer.subjects.recovery` - `string`

Email subject to use for password reset. Defaults to `Reset Your Password`.

`mailer.subjects.email_change` - `string`

Email subject to use for email change confirmation. Defaults to `Confirm Email Change`.

`mailer.templates.confirmation` - `string`

Email template when confirming a signup.
`SiteURL`, `Email`, and `ConfirmationURL` variables are available.

Default:
```html
<h2>Confirm your signup</h2>

<p>Follow this link to confirm your account:</p>
<p><a href="{{ .ConfirmationURL }}">Confirm your mail</a></p>
```

`mailer.templates.recovery` - `string`

Email template when resetting a password.
`SiteURL`, `Email`, and `ConfirmationURL` variables are available.

Default:
```html
<h2>Reset Password</h2>

<p>Follow this link to reset the password for your account:</p>
<p><a href="{{ .ConfirmationURL }}">Reset Password</a></p>
```

`mailer.templates.email_change` - `string`

Email template when confirming the change of an email address.
`SiteURL`, `Email`, `NewEmail`, and `ConfirmationURL` variables are available.

Default:
```html
<h2>Confirm Change of Email</h2>

<p>Follow this link to confirm the update of your email from {{ .Email }} to {{ .NewEmail }}:</p>
<p><a href="{{ .ConfirmationURL }}">Change Email</a></p>
```


## Endpoints

GoTrue exposes the following endpoints:

* **POST /signup**

  Register a new user with an email and password.

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

* **POST /verify**

  Verify a registration or a password recovery. Type can be `signup` or `recover`
  and the `token` is a token returned from either `/signup` or `/recover`.

  ```json
  {
    "type": "signup",
    "token": "confirmation-code-delivered-in-email"
  }
  ```

  Returns:

  ```json
  {
    "access_token": "jwt-token-representing-the-user",
    "token_type": "bearer",
    "expires_in": 3600,
    "refresh_token": "a-refresh-token"
  }
  ```

* **POST /recover**

  Password recovery. Will deliver a password recovery mail to the user based on
  email address.

  ```json
  {
    "email": "email@example.com"
  }
  ```

  Returns:

  ```json
  {}
  ```

* **POST /token**

  This is an OAuth2 endpoint that currently implements
  the password and refresh_token grant types

  ```
  grant_type=password&username=email@example.com&password=secret
  ```

  or

  ```
  grant_type=refresh_token&refresh_token=my-refresh-token
  ```

  Once you have an access token, you can access the methods requiring authentication
  by settings the `Authorization: Bearer YOUR_ACCESS_TOKEN_HERE` header.

  Returns:

  ```json
  {
    "access_token": "jwt-token-representing-the-user",
    "token_type": "bearer",
    "expires_in": 3600,
    "refresh_token": "a-refresh-token"
  }
  ```

* **GET /user**

  Get the JSON object for the logged in user (requires authentication)

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

* **PUT /user**

  Update a user (Requires authentication). Apart from changing email/password, this
  method can be used to set custom user data.

  ```json
  {
    "email": "new-email@example.com",
    "password": "new-password",
    "data": {
      "key": "value",
      "number": 10,
      "admin": false
    }
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

* **POST /logout**

  Logout a user (Requires authentication).

  This will revoke all refresh tokens for the user. Remember that the JWT tokens
  will still be valid for stateless auth until they expires.


## TODO

* Schema for custom user data in config file
