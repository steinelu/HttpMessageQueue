package main

import (
    "fmt"
    "net/http"
    "log"
    "flag"
    "io"
    "strings"
    "sync"
)


type Topic struct {
    lock sync.Mutex
    subscribers map[*http.ResponseWriter]struct{} // maps from subscriber to its current pos in the queue
}


func NewTopic() (topic *Topic){
    topic = new(Topic)
    //topic.messageQueue = make([][]bytes)
    topic.subscribers = make(map[*http.ResponseWriter]struct{})
    return
}

func (t *Topic) broadcastMessage(msg []byte){
    fmt.Println("broadcastMessage")
    

    for w,_ := range t.subscribers{
        flusher, ok := (*w).(http.Flusher)
        if !ok {
            panic("expected http.ResponseWriter to be an http.Flusher")
        }
        fmt.Printf("%s", msg)
        (*w).Write(msg)
        (*w).Write([]byte("\n"))
        flusher.Flush()
    }
}

func (t *Topic) addSubscriber(w *http.ResponseWriter){
    fmt.Println("addSub")
    t.lock.Lock()
    t.subscribers[w] = struct{}{}
    t.lock.Unlock()
}

func (t *Topic) removeSubscriber(w *http.ResponseWriter){
    fmt.Println("removeSub")
    t.lock.Lock()
    _, ok := t.subscribers[w]
    if ok{
        delete(t.subscribers, w)
    }
    t.lock.Unlock()
}

func (t Topic) Print(){
    fmt.Printf("%d subscribers\n", len(t.subscribers))
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

type Server struct{
    queue map[string]*Topic
}

func (s Server)PrintQueue(){
    fmt.Printf("len(queue) = %d\n", len(s.queue))
    for k, t := range s.queue { 
        fmt.Printf("%v -> %d\n", k, len(t.subscribers))
    }
}

//Subscriber
func (s *Server) handleGet(w http.ResponseWriter, req *http.Request){
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    w.WriteHeader(http.StatusOK)

    flusher, ok := w.(http.Flusher)
    if !ok {
        panic("expected http.ResponseWriter to be an http.Flusher")
    }
    flusher.Flush()

    topic, ok := s.queue[req.URL.String()]

    if !ok {
        topic = NewTopic()
        s.queue[req.URL.String()] = topic
    }

    topic.addSubscriber(&w)
    notify := w.(http.CloseNotifier).CloseNotify()
    
    select{
    case <- notify:
        fmt.Println("Notify")
        return
    case <- req.Context().Done():
        fmt.Println("Context.Done")
        return
    }
    
    topic.removeSubscriber(&w)
    //queue[req.URL.String()] = topic
}


//Publisher
func (s *Server)handlePost(w http.ResponseWriter, req *http.Request){
    body, err := io.ReadAll(req.Body)

    if err != nil {
        log.Fatalln(err)
    }

    topic, ok := s.queue[req.URL.String()]
    if !ok {
        topic = NewTopic()
    }
    
    topic.broadcastMessage(body)
    //s.queue[req.URL.String()] = topic
}

func main() {
    addr := flag.String("addr", "127.0.0.1:8080", "proxy's listening address")
    //toAddr := flag.String("to", "127.0.0.1:8080", "the address this proxy will forward to")
    flag.Parse()

    server := Server{make(map[string]*Topic)}

    http.HandleFunc("/",
        func(w http.ResponseWriter, req *http.Request) {
            switch req.Method {
            case "GET":
                server.handleGet(w, req)

            case "POST":
                server.handlePost(w, req)
            }
            fmt.Println()
            server.PrintQueue()
        })

    log.Println("Starting server on", *addr)
    if err := http.ListenAndServe(*addr, nil); err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}