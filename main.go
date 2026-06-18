package main
import (
    "fmt"
    "net/http"
    "log"
)
func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Service is running and healthy")
}
func main() {
    http.HandleFunc("/health", handler)
    log.Println("Server starting on port 8080...")
    log.Fatal(http.ListenAndServe(":8080", nil))
}