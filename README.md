# Swaggor

## Service Overview
Swaggor is supposed to be a swagger generator for Golang, while [echo](https://github.com/labstack/echo) acts as a web service.
The idea is to add a comment like **"//SWAGGOR"** above the echo endpoint definition. Swaggor should generate a swagger document for that endpoint. Swagger documents must include the following:
- Description (can be found in the handler method comment or a comment above the endpoint definition.)
- Path
- Method
- Including all possible http status codes and their contents in the response.

**As of right now, this project is still in the construction phase and is not fully functional.**

## Usage
Usage supposed to be like this:
```go
groupV2 := v2.Group("/group", 
		tracing.MiddlewareWrapper("SomeMiddleware", 
		middleware.SomeMiddleware(middleware.RestrictPassengerAuthMode)))
	
// SWAGGOR
groupV2.GET("/something", tracing.HandlerWrapper("EndpointHandler",
		controller.EndpointHandler(dbRepo, msgRepo)),
		tracing.MiddlewareWrapper("SomeMiddleware", 
		middleware.SomeMiddleware(middleware.SomeEnum)))
```
