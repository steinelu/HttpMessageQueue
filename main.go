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
    lock sync.RWMutex
    subscribers map[*http.ResponseWriter]*sync.Mutex // maps from subscriber to its current pos in the queue
}


func NewTopic() (topic *Topic){
    topic = new(Topic)
    topic.subscribers = make(map[*http.ResponseWriter]*sync.Mutex)
    return
}


func (t *Topic) broadcastMessage(msg []byte){
    //fmt.Println("broadcastMessage")
    t.lock.RLock()
    for w, l := range t.subscribers {
        go func(w *http.ResponseWriter, l *sync.Mutex){
            flusher, ok := (*w).(http.Flusher)
            if !ok {
                panic("expected http.ResponseWriter to be an http.Flusher")
            }
            
            l.Lock()
            fmt.Fprintf(*w, "%x\r\n%s\r\n", len(msg), msg)

            //size := len(msg)
            //(*w).Write([]byte(fmt.Sprintf("%X\r\n", size)))
            //(*w).Write(msg)
            //(*w).Write([]byte("\r\n"))
            flusher.Flush()
            l.Unlock()
        }(w, l)
    }
    t.lock.RUnlock()
}

func (t *Topic) addSubscriber(w *http.ResponseWriter){
    fmt.Println("addSub")
    t.lock.Lock()
    t.subscribers[w] = &sync.Mutex{}
    t.lock.Unlock()
}

func (t *Topic) removeSubscriber(w *http.ResponseWriter){
    fmt.Println("removeSub")
    t.lock.RLock()
    _, ok := t.subscribers[w]
    if ok{
        delete(t.subscribers, w)
    }
    t.lock.RUnlock()
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
    lock sync.RWMutex
    mux *http.ServeMux
}

func NewServer() (server *Server){
    server = new(Server)
    server.mux = http.NewServeMux()
    server.queue = make(map[string]*Topic)
    server.routes()
    return
}


func Info(h http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request){
        clientIp := req.Header.Get("X-FORWARDED-FOR")
	    
        fmt.Printf("+ (%v) (%v) [%s] %s\n", clientIp, req.RemoteAddr, req.Method, req.URL.String())
        h(w, req)
        //fmt.Printf("- [%s] %s\n", req.Method, req.URL.String())
    }
}


func (server *Server) routes(){
    server.mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request){
        switch req.Method {
        case "GET":
            //server.handlePost(w, req)
            Info(server.handleGet)(w, req)
        case "POST":
            Info(server.handlePost)(w, req)
        }
        fmt.Println()
        server.PrintQueue()
    })
    //s.mux.HandleFunc("POST /one", s.handlePost)
    //s.mux.HandleFunc("GET /one", s.handleGet)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request){
    s.mux.ServeHTTP(w, req)
}

func (s *Server)PrintQueue(){
    s.lock.RLock()
    defer s.lock.RUnlock()

    fmt.Printf("[%d] ", len(s.queue))
    for k, t := range s.queue { 
        fmt.Printf("[%d]%s", len(t.subscribers), k)
    }
    //fmt.Println()
}

//Subscriber
func (s *Server) handleGet(w http.ResponseWriter, req *http.Request){
    //fmt.Println("HandleGet")
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    w.WriteHeader(http.StatusOK)

    flusher, ok := w.(http.Flusher)
    if !ok {
        panic("expected http.ResponseWriter to be an http.Flusher")
    }
    flusher.Flush()

    name := req.URL.String()
    //name :=  req.PathValue("topic")

    s.lock.RLock()
    topic, ok := s.queue[name]
    s.lock.RUnlock()

    if !ok {
        topic = NewTopic()
        s.lock.Lock()
        s.queue[name] = topic
        s.lock.Unlock()
    }

    topic.addSubscriber(&w)
    s.PrintQueue()
    notify := w.(http.CloseNotifier).CloseNotify()
    
    select{
    case <- notify:
        fmt.Println("Notify")
    case <- req.Context().Done():
        fmt.Println("Context.Done")
    }
    
    topic.removeSubscriber(&w)

    topic.lock.RLock()
    if len(topic.subscribers) <= 0 {
        s.lock.Lock()
        delete(s.queue, name)
        s.lock.Unlock()
    }
    topic.lock.RUnlock()
}


//Publisher
func (s *Server)handlePost(w http.ResponseWriter, req *http.Request){
    //fmt.Println("HandlePost")
    body, err := io.ReadAll(req.Body)

    if err != nil {
        log.Fatalln(err)
    }
    name := req.URL.String()
    //name :=  req.PathValue("topic")

    s.lock.RLock()
    topic, ok := s.queue[name]
    s.lock.RUnlock()

    if !ok {
        topic = NewTopic()
        s.lock.Lock()
        s.queue[name] = topic
        s.lock.Unlock()
    }
    
    go topic.broadcastMessage(body)
}

func main() {
    addr := flag.String("addr", "127.0.0.1:8080", "proxy's listening address")
    flag.Parse()

    server := NewServer()

    log.Println("Starting server on", *addr)
    if err := http.ListenAndServe(*addr, server); err != nil {
        log.Fatal("ListenAndServe:", err)
    }
}