# TAOS 

## Getting Started

### Installation

If this is your first time with golang, please follow [the instructions](https://golang.org/doc/install) to install (requires golang 1.5 or above).

After installing golang:

```shell
# Install taos
go get github.com/kmacoskey/taos

# Taos uses glide to vendor dependencies.
# Install glide
go get -u github.com/Masterminds/glide

# Resolve dependencies with glide
cd $GOPATH/kmacoskey/taos
make depends
```

Next, create a PostgreSQL database (not covered here) and set connection information in the configuration file `config/app.yaml`, or set the environment variable `RESTFUL_DSN` like the following:

```
postgres://<username>:<password>@<server-address>:<server-port>/<db-name>
```

Refer to the [pq package GoDoc](https://godoc.org/github.com/lib/pq) for supported connection paramaters.

### Test

Ensure successfull installation by running the tests:

```shell
make test
```

### Run the Server

Use the default make target:

```shell
make
```

or directly with the go CLI:

```shell
go run server.go
```

The application starts an HTTP server at the default port of 8080. 
TODO Reference where routes can be found

### Client Usage

Taos currently implements basic auth with JSON Web Tokens. Authenticate first to receive a token:

```shell
# Authenticate with development credentials
curl -X POST -H "Content-Type: application/json" -d '{"username": "demo", "password": "pass"}' http://localhost:8080/v1/auth
# Returns a JWT:
{"token":"<TOKEN CONTENTS>"}
```

Using the above token after successful authentication:

```shell
# With a valid JWT
curl -X GET -H "Authorization: Bearer <TOKEN CONTENTS>" http://localhost:8080/v1/clusters
```

* `models/album.go`: contains the data structure representing a row in the new table.
* `services/album.go`: contains the business logic that implements the CRUD operations.
* `daos/album.go`: contains the DAO (Data Access Object) layer that interacts with the database table.
* `apis/album.go`: contains the API layer that wires up the HTTP routes with the corresponding service APIs.

## Code Structure

* `apis`: contains the API layer that wires up the HTTP routes with the corresponding service APIs
* `services`: contains the main business logic of the application
* `daos`: contains the DAO (Data Access Object) layer that interacts with persistent storage
* `models`: contains the data structures used for communication between different layers
* `app`: contains routing middlewares and application-level configurations
* `errors`: contains error representation and handling

The main entry of the application is in the `server.go` file. It does the following work:

* load external configuration
* establish database connection
* instantiate components and inject dependencies
* start the HTTP server
