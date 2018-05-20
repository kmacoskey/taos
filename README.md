# TAOS 

## Getting Started

### Installation

If this is your first time with golang, please follow [the instructions](https://golang.org/doc/install) to install (requires golang 1.8 or above).

After installing golang:

```shell
# Install taos
go get github.com/kmacoskey/taos
```

### Configuration

Copy the example configuration file:

```shell
cp config.example.yml config.yml
```

Next, create a PostgreSQL database (not covered here) and set connection information in the configuration file `config.yaml`:

```
conn_str: "postgres://<role>:<password>@<host>:<port>/<database>?sslmode=disable"
```

Refer to the [pq package GoDoc](https://godoc.org/github.com/lib/pq) for supported connection paramaters.

### Test

Ensure successfull installation by running the tests:

```shell
make test
```

### Run the Server

```shell
make
make run
```

The application starts an HTTP server at the default port of 8080. 

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
