# Device Flow Proxy
![CI](https://github.com/wraix/device-flow-proxy/actions/workflows/ci.yml/badge.svg)

A demonstration of the OAuth 2.0 Device Code flow for devices without a browser or with limited keyboard entry.

This service acts as an OAuth server that implements the device code flow, proxying to a real OAuth server behind the scenes.

Original implementation in PHP by [Aaron Parecki](https://github.com/aaronpk) can be found here: [Device Flow Proxy Server](https://github.com/aaronpk/Device-Flow-Proxy-Server).

## Important Notice

This is not ready for production, it serves as an example. Use at your own risk.

## Features

- [x] Device Authorization Grant (rfc8628)
- [x] Automated Documentation of Endpoints using OpenAPI @ /docs
- [x] Metrics endpoint using Prometheus @ /metrics
- [x] Tracing with OpenTelemetry and Jaeger
- [x] In memory storage of tokens
- [x] Example configuration for Ory Hydra

## Requirements

- OAuth 2 Provider with support for Authorization Code

## Getting Started with Device Flow Proxy & Ory Hydra

To get started with `Device Authorization Grant` using the Device Flow Proxy, an OAuth 2.0 provider capable of performing `Authorization Code` flow is required, preferably with PKCE.

This examples uses an open source OAuth2 Provider from [Ory](https://www.ory.sh) named [Hydra](https://www.ory.sh/hydra/)

To get hydra up an running please refer to the documentation. The 5 minute tutorial should be sufficent for example purpose, see https://www.ory.sh/hydra/docs/5min-tutorial/. Use the quickstart guide with hydra, postgres and distributed tracing using jaeger:

```
docker-compose -f quickstart.yml \
    -f quickstart-postgres.yml \
    -f quickstart-tracing.yml \
    up --build
```

Getting the device flow proxy up and running with hydra is as simple as performing a checkout and starting the container:

```
git clone git@github.com:wraix/device-flow-proxy.git
cd device-flow-proxy
docker-compose up
```

A client for the device is required to be registered in Hydra. Often devices will be public clients, so please require PKCE. Client registration is probably dynamic, so an option could be to use dynamic client registration if the feature is supported by the OAuth 2.0 Provider. For this example you will simply register the client manually at Hydra. You'll need to set the proxy's URL as the callback URL in the OAuth application registration and ensure that the grant_types `authorization_code` and `urn:ietf:params:oauth:grant-type:device_code` is added to the client, usually you want the device to be able to use refresh_tokens so add the grant type `refresh_token` as well:

```
docker-compose -f quickstart.yml exec hydra \
    hydra clients create \
    --endpoint http://127.0.0.1:4445/ \
    --id 82a3d148-e386-44b5-9761-ffcfdf58b84c \
    --grant-types authorization_code,refresh_token,urn:ietf:params:oauth:grant-type:device_code \
    --token-endpoint-auth-method none \
    --callbacks http://localhost:8080/auth/redirect
```

The device can begin the flow by making a POST request to this proxy:

```
curl http://localhost:8080/device/code -d client_id=82a3d148-e386-44b5-9761-ffcfdf58b84c
```

The response will contain the URL the user should visit and the code they should enter, as well as a long device code.

```json
{
	"device_code": "edae0f198e83e908882c1482b99f30be09e1334669c86e266f40997bae7ffde3",
	"verification_uri": "http://localhost:8080/device",
	"user_code": "F8AH-0KPB",
	"expires_in": 300,
	"interval": 5
}
```

The device should instruct the user to visit the URL and enter the code, or can provide a full link that pre-fills the code for the user in case the device is displaying a QR code.

```
http://localhost:8080/device?code=F8AH-0KPB
```

The device should then poll the token endpoint at the interval provided, making a POST request like the below:

```
curl http://localhost:8080/device/token -d grant_type=urn:ietf:params:oauth:grant-type:device_code \
  -d client_id=82a3d148-e386-44b5-9761-ffcfdf58b84c \
  -d device_code=33e50b3717bd1ab4de0303a549da040a5df2bab1f85d5d4cc27750e0725dd72c
```

While the user is busy logging in, the response will be

```json
{"error":"authorization_pending"}
```

Once the user has finished logging in and granting access to the application, the response will contain an access token.

```
{
	"access_token": "3yvX87sNjFUxgSVB72oleIRyzfdC3yUL5urKoR0tLwI.gXc4NSfiSpBzksfagVuZw63XCjsmR7I2jXaJ7OBZj5c",
	"expires_in": 3599,
	"scope": "",
	"token_type": "bearer"
}
```

The device can now use the access token to access resource servers. To introspect the access token in hydra use

```
docker-compose -f quickstart.yml exec hydra \
    hydra token introspect \
    --endpoint http://127.0.0.1:4445/ \
    3yvX87sNjFUxgSVB72oleIRyzfdC3yUL5urKoR0tLwI.gXc4NSfiSpBzksfagVuZw63XCjsmR7I2jXaJ7OBZj5c
```

Response should look something like:

```json
{
	"active": true,
	"aud": [],
	"client_id": "82a3d148-e386-44b5-9761-ffcfdf58b84c",
	"exp": 1636115113,
	"iat": 1636111513,
	"iss": "http://127.0.0.1:4444/",
	"nbf": 1636111513,
	"sub": "foo@bar.com",
	"token_type": "Bearer",
	"token_use": "access_token"
}
```

Enjoy.
