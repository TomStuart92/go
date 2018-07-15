
https://github.com/cdimascio/go-bunyan-logger

clone

run go get ./.. -t to install all dependencies




cassandra

follow guide https://gist.github.com/hkhamm/a9a2b45dd749e5d3b3ae
cqlsh
CREATE KEYSPACE tom WITH REPLICATION = { 'class': 'SimpleStrategy', 'replication_factor': 1 };
CREATE TABLE tom.test (id text PRIMARY KEY, body text);
SELECT * FROM tom.test; (To test)

logLevel=fatal go test
go run main.go | bunyan