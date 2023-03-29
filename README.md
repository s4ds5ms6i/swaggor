# Swaggor

## Service Overview
Swaggor is an OpenAPI ([swagger](https://swagger.io/)) generator for Golang, while [echo](https://github.com/labstack/echo) acts as a web service.
The idea is to add a comment like **"//SWAGGOR"** above the echo endpoint definition. Swaggor should generate a swagger document for that endpoint. Swagger documents must include the following:
- Description (can be found in the handler method comment or a comment above the endpoint definition.)
- Path
- Method
- Including all possible http status codes and their contents in the response.

## Usage
Usage is like this:\
\
You add the SWAGGOR tag above your endpoint declaration:
```go
groupV2 := v2.Group("/group",
tracing.MiddlewareWrapper("SomeMiddleware",
middleware.SomeMiddleware(middleware.SomeEnum)))

// SWAGGOR
groupV2.GET("/something", tracing.HandlerWrapper("EndpointHandler",
controller.EndpointHandler(dbRepo, msgRepo)),
tracing.MiddlewareWrapper("SomeMiddleware",
middleware.SomeMiddleware(middleware.SomeEnum)))
```
Then you run a CLI command to generate OpenAPI:
```bash
swaggor -r <project directory> [-e <directories to be excluded in csv form>]
```
\
**As of right now, this project is still in the construction phase and is not fully functional.**
