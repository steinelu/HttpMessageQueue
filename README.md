# HTTP PubSub System

A server that receives messages on a topic distributes them to the clients requesting the messages of the topic.

## Run the Server

```shell
go run .
```


## Receive Message
Receive messages form the topic '/example'

```shell
curl <domain>/example

curl localhost:8080/example
```


## Send Message
Send a message to a topic '/example'

```shell
curl <domain>/example -X POST -d "<message>"

curl localhost:8080/example -X POSt -d "Hello World"
```

### Possible Future Features
- Controlls
    - Gracefulll shutdown
    - change parameters at run time (config-ish)
    - heartbeats
- Monitoring
- Logging
- Configurable message cache limitation
    - auto delete 
    - timestamp
- Rate limiting
    - Bandwidth
    - num connections
- Authorization
    - Users
    - Access tokens
- Dristributed - Scaling
- Encryption
