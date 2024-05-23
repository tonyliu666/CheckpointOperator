package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

type Event struct {
    Events []struct {
        ID        string    `json:"id"`
        Timestamp time.Time `json:"timestamp"`
        Action    string    `json:"action"`
        Target    struct {
            MediaType  string `json:"mediaType"`
            Size       int    `json:"size"`
            Digest     string `json:"digest"`
            Length     int    `json:"length"`
            Repository string `json:"repository"`
            URL        string `json:"url"`
        } `json:"target"`
        Request struct {
            ID        string `json:"id"`
            Addr      string `json:"addr"`
            Host      string `json:"host"`
            Method    string `json:"method"`
            UserAgent string `json:"userAgent"`
        } `json:"request"`
        Actor struct {
            Name string `json:"name"`
        } `json:"actor"`
        Source struct {
            Addr       string `json:"addr"`
            InstanceID string `json:"instanceID"`
        } `json:"source"`
    } `json:"events"`
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
    var event Event

    // Decode the JSON payload from the request body
    if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        http.Error(w, "failed to decode request body", http.StatusBadRequest)
        fmt.Printf("Error decoding JSON: %v\n", err)
        return
    }

    // Print the event for debugging purposes
    fmt.Printf("Received event: %+v\n", event)
    fmt.Println("Repository:", event.Events[0].Target.Repository)

    // Respond with HTTP 200 OK status
    w.WriteHeader(http.StatusOK)
}

func main() {
    http.HandleFunc("/webhook", webhookHandler)
    fmt.Println("Server started at :8080")
    http.ListenAndServe(":8080", nil)
}
