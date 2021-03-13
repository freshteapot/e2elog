# End to end testing log
- I want to write end to end tests, that give me some belief I am covering my endpoints
- Consumes an openapi version 3 spec
- Consumes ndjson file of logs.

# Log format
```json
{
    "method":"POST",
    "url":"/api/v1/user/register",
    "status_code":201
}
```


## Example

### All
```sh
cd example
go run main.go
```

### Coverage only
```sh
cd example
go run main.go -coverage
```

### Stats
```sh
cd example
go run main.go -coverage
```



# Reference
- https://github.com/getkin/kin-openapi/blob/master/pathpattern/node_test.go
