# Message Queue

A server that receives messages on a topic, buffers the messages and sends them to an client requesting the messages of the specified topic.

## Run the Server

```shell
go run .
```

## Send Message
Send a message to a topic '/example'

```shell
curl <domain>/example

curl localhost:8080/example
```

## Receive Message
Receive messages form the topic '/example'
```shell
curl <domain>/example -X POST -d "<message>"

curl localhost:8080/example -X POSt -d "Hello World"
```

### Possible Future Features
- Configurable message cache limitation
    - auto delete 
    - timestamp
- Rate limiting
- Authorization
    - Users
    - Access tokens
- Dristributed
