package main

import (
    "fmt"
    "net/http"
    "log"
    "flag"
    "io"
    "strings"
)

type void struct{}

type Topic struct {
    messages [][]byte
    subscribers map[chan []byte]void
}

func (t *Topic) addMessage(msg []byte){
    fmt.Println("addMessage")
    t.messages = append(t.messages, msg)
}

func (t *Topic) addSubscriber(c chan []byte){
    fmt.Println("addSub")
    _, ok := t.subscribers[c]
    if !ok{
        t.subscribers[c] = void{}
    }
    fmt.Println("END addSub")
}

func (t *Topic) removeSubscriber(c chan []byte){
    fmt.Println("removeSub")
    _, ok := t.subscribers[c]
    if ok{
        delete(t.subscribers, c)
    }
}

func (t Topic) Print(){
    fmt.Printf("|subs| = %d\t|msgs| = % d\n", len(t.subscribers), len(t.messages))
}

//type Queue []Topic


func PrintQueue(queue map[string]*Topic){
    fmt.Printf("len(queue) = %d\n", len(queue))
    for k, v := range queue { 
        fmt.Printf("\t%v -> %v\n", k, v)
    }
}

func PrintHeader(req *http.Request){
    var b strings.Builder

    fmt.Fprintf(&b, "%v\t%v\t%v\tHost: %v\n", req.RemoteAddr, req.Method, req.URL, req.Host)
    for name, headers := range req.Header {
        for _, h := range headers {
            fmt.Fprintf(&b, "%v: %v\n", name, h)
        }
    }
    log.Println(b.String())
}


//Subscriber
func handleGet(w http.ResponseWriter, req *http.Request, queue map[string]*Topic){
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        panic("expected http.ResponseWriter to be an http.Flusher")
    }

    topic, ok := queue[req.URL.String()]

    if !ok {
        return
    }

    for i := 0; i < len(topic.messages); i++ {
        //w.Write(topic.messages[i])
        fmt.Fprintf(w, "%s\n", topic.messages[i])
    }

    flusher.Flush()

    done := make(chan bool)
    push := make(chan []byte)

    topic.addSubscriber(push)
    queue[req.URL.String()] = topic

    //https://stackoverflow.com/questions/51945810/how-to-non-destructively-check-if-an-http-client-has-shut-down-the-connection
    //http://technosophos.com/2013/12/11/go-get-notified-when-an-http-request-closes.html

    notify := w.(http.CloseNotifier).CloseNotify()

    go func(){
        for {
            select{
            case msg := <- push:
                //fmt.Println("New Message")
                _, ok := w.Write(msg)
                if ok != nil {
                    done <- false
                    return
                }
                _, ok = w.Write([]byte("\n"))
                if ok != nil {
                    done <- false
                    return
                }
                flusher.Flush()
                //fmt.Println("New Message Flush")
            case <- notify:
                fmt.Println("Notify")
                done <- false
                return
            case <- req.Context().Done():
                fmt.Println("Context.Done")
                done <- false
                return
            }
        }
    }()
    <- done
    topic.removeSubscriber(push)
    queue[req.URL.String()] = topic
}


//Publisher
func handlePost(w http.ResponseWriter, req *http.Request, queue map[string]*Topic){
    body, err := io.ReadAll(req.Body)

    if err != nil {
        log.Fatalln(err)
    }

    topic, ok := queue[req.URL.String()]
    if !ok {
        topic = &Topic{subscribers:make(map[chan []byte]void)}
    }
    
    topic.Print()
    topic.addMessage(body)
    topic.Print()
    queue[req.URL.String()] = topic
    PrintQueue(queue)
    

    //for i := 0; i < len(topic.subscribers); i++ {
    for sub, _ := range topic.subscribers{
        sub <- body
    }
}

func main() {
    addr := flag.String("addr", "127.0.0.1:8080", "proxy's listening address")
    //toAddr := flag.String("to", "127.0.0.1:8080", "the address this proxy will forward to")
    flag.Parse()

    queue := make(map[string]*Topic)

    http.HandleFunc("/",
        func(w http.ResponseWriter, req *http.Request) {
            //PrintQueue(queue)
            switch req.Method {
            case "GET":
                handleGet(w, req, queue)

            case "POST":
                handlePost(w, req, queue)
            }
            //PrintQueue(queue)
            fmt.Println()
        })

    //addr := ":8080"
    log.Println("Starting server on", *addr)
    if err := http.ListenAndServe(*addr, nil); err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}