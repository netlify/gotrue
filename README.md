![GoTrue](gotrue.png)

<p align="center">User management for APIs</p>

GoTrue is a small open-source API written in Golang, that can act as a self-standing
API service for handling user registration and authentication for Jamstack projects.

It's based on OAuth2 and JWT and will handle user signup, authentication and custom
user data.

## Configuration

You may configure GoTrue using either a configuration file named `.env`,
environment variables, or a combination of both. Environment variables are prefixed with `GOTRUE_`, and will always have precedence over values provided via file.

### Top-Level

```properties
GOTRUE_SITE_URL=https://example.netlify.com/
```

`SITE_URL` - `string` **required**

The base URL your site is located at. Currently used in combination with other settings to construct URLs used in emails.

`OPERATOR_TOKEN` - `string` _Multi-instance mode only_

The shared secret with an operator (usually Netlify) for this microservice. Used to verify requests have been proxied through the operator and
the payload values can be trusted.

`DISABLE_SIGNUP` - `bool`

When signup is disabled the only way to create new users is through invites. Defaults to `false`, all signups enabled.

`GOTRUE_RATE_LIMIT_HEADER` - `string`

Header on which to rate limit the `/token` endpoint.

### API

```properties
GOTRUE_API_HOST=localhost
PORT=9999
```

`API_HOST` - `string`

Hostname to listen on.

`PORT` (no prefix) / `API_PORT` - `number`

Port number to listen on. Defaults to `8081`.

`API_ENDPOINT` - `string` _Multi-instance mode only_

Controls what endpoint Netlify can access this API on.

`REQUEST_ID_HEADER` - `string`

If you wish to inherit a request ID from the incoming request, specify the name in this value.

### Database

```properties
GOTRUE_DB_DRIVER=mysql
DATABASE_URL=root@localhost/gotrue
```

`DB_DRIVER` - `string` **required**

Chooses what dialect of database you want. Must be `mysql`.

`DATABASE_URL` (no prefix) / `DB_DATABASE_URL` - `string` **required**

Connection string for the database.

`DB_NAMESPACE` - `string`

Adds a prefix to all table names.

**Migrations Note**

Migrations are not applied automatically, so you will need to run them after
you've built gotrue.

* If built locally: `./gotrue migrate`
* Using Docker: `docker run --rm gotrue gotrue migrate`

### Logging

```properties
LOG_LEVEL=debug # available without GOTRUE prefix (exception)
GOTRUE_LOG_FILE=/var/log/go/gotrue.log
```

`LOG_LEVEL` - `string`

Controls what log levels are output. Choose from `panic`, `fatal`, `error`, `warn`, `info`, or `debug`. Defaults to `info`.

`LOG_FILE` - `string`

If you wish logs to be written to a file, set `log_file` to a valid file path.

### Opentracing
Currently, only the Datadog tracer is supported.

```properties
GOTRUE_TRACING_ENABLED=true
GOTRUE_TRACING_HOST=127.0.0.1
GOTRUE_TRACING_PORT=8126
GOTRUE_TRACING_TAGS="tag1:value1,tag2:value2"
GOTRUE_SERVICE_NAME="gotrue"
```

`TRACING_ENABLED` - `bool`

Whether tracing is enabled or not. Defaults to `false`.

`TRACING_HOST` - `bool`

The tracing destination.

`TRACING_PORT` - `bool`

The port for the tracing host.

`TRACING_TAGS` - `string`

A comma separated list of key:value pairs. These key value pairs will be added as tags to all opentracing spans.

`SERVICE_NAME` - `string`

The name to use for the service.

### JSON Web Tokens (JWT)

```properties
GOTRUE_JWT_SECRET=supersecretvalue
GOTRUE_JWT_EXP=3600
GOTRUE_JWT_AUD=netlify
```

`JWT_SECRET` - `string` **required**

The secret used to sign JWT tokens with.

`JWT_EXP` - `number`

How long tokens are valid for, in seconds. Defaults to 3600 (1 hour).

`JWT_AUD` - `string`

The default JWT audience. Use audiences to group users.

`JWT_ADMIN_GROUP_NAME` - `string`

The name of the admin group (if enabled). Defaults to `admin`.

`JWT_DEFAULT_GROUP_NAME` - `string`

The default group to assign all new users to.

### External Authentication Providers

We support `bitbucket`, `github`, `gitlab`, and `google` for external authentication.
Use the names as the keys underneath `external` to configure each separately.

```properties
GOTRUE_EXTERNAL_GITHUB_CLIENT_ID=myappclientid
GOTRUE_EXTERNAL_GITHUB_SECRET=clientsecretvaluessssh
```

No external providers are required, but you must provide the required values if you choose to enable any.

`EXTERNAL_X_ENABLED` - `bool`

Whether this external provider is enabled or not

`EXTERNAL_X_CLIENT_ID` - `string` **required**

The OAuth2 Client ID registered with the external provider.

`EXTERNAL_X_SECRET` - `string` **required**

The OAuth2 Client Secret provided by the external provider when you registered.

`EXTERNAL_X_REDIRECT_URI` - `string` **required for gitlab**

The URI a OAuth2 provider will redirect to with the `code` and `state` values.

`EXTERNAL_X_URL` - `string`

The base URL used for constructing the URLs to request authorization and access tokens. Used by `gitlab` only. Defaults to `https://gitlab.com`.

### E-Mail

Sending email is not required, but highly recommended for password recovery.
If enabled, you must provide the required values below.

```properties
GOTRUE_SMTP_HOST=smtp.mandrillapp.com
GOTRUE_SMTP_PORT=587
GOTRUE_SMTP_USER=smtp-delivery@example.com
GOTRUE_SMTP_PASS=correcthorsebatterystaple
GOTRUE_SMTP_ADMIN_EMAIL=support@example.com
GOTRUE_MAILER_SUBJECTS_CONFIRMATION="Please confirm"
```

`SMTP_ADMIN_EMAIL` - `string` **required**

The `From` email address for all emails sent.

`SMTP_HOST` - `string` **required**

The mail server hostname to send emails through.

`SMTP_PORT` - `number` **required**

The port number to connect to the mail server on.

`SMTP_USER` - `string`

If the mail server requires authentication, the username to use.

`SMTP_PASS` - `string`

If the mail server requires authentication, the password to use.

`SMTP_MAX_FREQUENCY` - `number`

Controls the minimum amount of time that must pass before sending another signup confirmation or password reset email. The value is the number of seconds. Defaults to 900 (15 minutes).

`MAILER_AUTOCONFIRM` - `bool`

If you do not require email confirmation, you may set this to `true`. Defaults to `false`.

`MAILER_URLPATHS_INVITE` - `string`

URL path to use in the user invite email. Defaults to `/`.

`MAILER_URLPATHS_CONFIRMATION` - `string`

URL path to use in the signup confirmation email. Defaults to `/`.

`MAILER_URLPATHS_RECOVERY` - `string`

URL path to use in the password reset email. Defaults to `/`.

`MAILER_URLPATHS_EMAIL_CHANGE` - `string`

URL path to use in the email change confirmation email. Defaults to `/`.

`MAILER_SUBJECTS_INVITE` - `string`

Email subject to use for user invite. Defaults to `You have been invited`.

`MAILER_SUBJECTS_CONFIRMATION` - `string`

Email subject to use for signup confirmation. Defaults to `Confirm Your Signup`.

`MAILER_SUBJECTS_RECOVERY` - `string`

Email subject to use for password reset. Defaults to `Reset Your Password`.

`MAILER_SUBJECTS_EMAIL_CHANGE` - `string`

Email subject to use for email change confirmation. Defaults to `Confirm Email Change`.

`MAILER_TEMPLATES_INVITE` - `string`

URL path to an email template to use when inviting a user.
`SiteURL`, `Email`, and `ConfirmationURL` variables are available.

Default Content (if template is unavailable):

```html
<h2>You have been invited</h2>

<p>You have been invited to create a user on {{ .SiteURL }}. Follow this link to accept the invite:</p>
<p><a href="{{ .ConfirmationURL }}">Accept the invite</a></p>
```

`MAILER_TEMPLATES_CONFIRMATION` - `string`

URL path to an email template to use when confirming a signup.
`SiteURL`, `Email`, and `ConfirmationURL` variables are available.

Default Content (if template is unavailable):

```html
<h2>Confirm your signup</h2>

<p>Follow this link to confirm your user:</p>
<p><a href="{{ .ConfirmationURL }}">Confirm your mail</a></p>
```

`MAILER_TEMPLATES_RECOVERY` - `string`

URL path to an email template to use when resetting a password.
`SiteURL`, `Email`, and `ConfirmationURL` variables are available.

Default Content (if template is unavailable):

```html
<h2>Reset Password</h2>

<p>Follow this link to reset the password for your user:</p>
<p><a href="{{ .ConfirmationURL }}">Reset Password</a></p>
```

`MAILER_TEMPLATES_EMAIL_CHANGE` - `string`

URL path to an email template to use when confirming the change of an email address.
`SiteURL`, `Email`, `NewEmail`, and `ConfirmationURL` variables are available.

Default Content (if template is unavailable):

```html
<h2>Confirm Change of Email</h2>

<p>Follow this link to confirm the update of your email from {{ .Email }} to {{ .NewEmail }}:</p>
<p><a href="{{ .ConfirmationURL }}">Change Email</a></p>
```

`WEBHOOK_URL` - `string`

Url of the webhook receiver endpoint. This will be called when events like `validate`, `signup` or `login` occur.

`WEBHOOK_SECRET` - `string`

Shared secret to authorize webhook requests. This secret signs the [JSON Web Signature](https://tools.ietf.org/html/draft-ietf-jose-json-web-signature-41) of the request. You *should* use this to verify the integrity of the request. Otherwise others can feed your webhook receiver with fake data.

`WEBHOOK_RETRIES` - `number`

How often GoTrue should try a failed hook.

`WEBHOOK_TIMEOUT_SEC` - `number`

Time between retries (in seconds).

`WEBHOOK_EVENTS` - `list`

Which events should trigger a webhook. You can provide a comma separated list.
For example to listen to all events, provide the values `validate,signup,login`.

## Endpoints

GoTrue exposes the following endpoints:

* **GET /settings**

  Returns the publicly available settings for this gotrue instance.

  ```json
  {
    "external": {
      "bitbucket": true,
      "github": true,
      "gitlab": true,
      "google": true
    },
    "disable_signup": false,
    "autoconfirm": false
  }
  ```

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
    "id": "11111111-2222-3333-4444-5555555555555",
    "email": "email@example.com",
    "confirmation_sent_at": "2016-05-15T20:49:40.882805774-07:00",
    "created_at": "2016-05-15T19:53:12.368652374-07:00",
    "updated_at": "2016-05-15T19:53:12.368652374-07:00"
  }
  ```

* **POST /invite**

  Invites a new user with an email.

  ```json
  {
    "email": "email@example.com"
  }
  ```

  Returns:

  ```json
  {
    "id": "11111111-2222-3333-4444-5555555555555",
    "email": "email@example.com",
    "confirmation_sent_at": "2016-05-15T20:49:40.882805774-07:00",
    "created_at": "2016-05-15T19:53:12.368652374-07:00",
    "updated_at": "2016-05-15T19:53:12.368652374-07:00",
    "invited_at": "2016-05-15T19:53:12.368652374-07:00"
  }
  ```

* **POST /verify**

  Verify a registration or a password recovery. Type can be `signup` or `recovery`
  and the `token` is a token returned from either `/signup` or `/recover`.

  ```json
  {
    "type": "signup",
    "token": "confirmation-code-delivered-in-email",
    "password": "12345abcdef"
  }
  ```

  `password` is required for signup verification if no existing password exists.

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
  the password, refresh_token, and authorization_code grant types

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
    "id": "11111111-2222-3333-4444-5555555555555",
    "email": "email@example.com",
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
    "id": "11111111-2222-3333-4444-5555555555555",
    "email": "email@example.com",
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
