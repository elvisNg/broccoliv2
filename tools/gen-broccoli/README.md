# gen-broccoli


## Usage

Define your service as `greeter.proto`

```
syntax = "proto3";

service Greeter {
	rpc Hello(Request) returns (Response) {}
}

message Request {
	string name = 1;
}

message Response {
	string msg = 1;
}
```

Generate the code

```
gen-broccoli --proto greeter.proto --dest ./
```
