package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

var (
	port                = "8080"
	currentTaskID       int
	currentSubscriberID int
	tasks               = make(map[int]*task)
)

func main() {
	http.HandleFunc("/start", handleStart)
	http.HandleFunc("/progress", handleTrackProgress)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sec, err := strconv.Atoi(r.URL.Query().Get("sec"))
	if err != nil {
		sec = 20
	}

	currentTaskID++
	id := currentTaskID
	task := &task{
		id: id,
		hub: &hub{
			subscribers:   make(map[int]*subscriber),
			unsubscribeCh: make(chan int),
		},
	}
	tasks[id] = task

	go task.run(sec)

	w.Write([]byte(strconv.Itoa(id) + "\n"))
}

func handleTrackProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	taskID, err := strconv.Atoi(r.URL.Query().Get("taskid"))
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	currentSubscriberID++
	sID := currentSubscriberID
	progressCh := make(chan *progress)
	doneCh := make(chan struct{})
	subscriber := &subscriber{
		id:         sID,
		progressCh: progressCh,
		doneCh:     doneCh,
	}
	task := tasks[taskID]
	task.addSubscriber(subscriber)

	setSSEHeaders(w)
	flusher := w.(http.Flusher)

	for {
		select {
		case prog := <-subscriber.progressCh:
			json.NewEncoder(w).Encode(prog)
			flusher.Flush()
		case <-subscriber.doneCh:
			return
		case <-r.Context().Done():
			task.unsubscribe(sID)
			return
		}
	}
}

type task struct {
	id    int
	Error error
	hub   *hub
}

type hub struct {
	subscribers   map[int]*subscriber
	unsubscribeCh chan int
}

type subscriber struct {
	id         int
	progressCh chan *progress
	doneCh     chan struct{}
}

type progress struct {
	Percent     int    `json:"percent"`
	Description string `json:"description"`
}

func (t *task) run(sec int) {
	c := 0
	for {
		select {
		case <-time.Tick(1 * time.Second):
			c++
			percent := 100 / sec * c
			t.updateProgress(percent, "increment")
			if c == sec {
				t.done()
				return
			}
		case id := <-t.hub.unsubscribeCh:
			t.unsubscribe(id)
		}
	}
}

func (t *task) updateProgress(percent int, desc string) {
	prog := &progress{Percent: percent, Description: desc}
	for _, subscriber := range t.hub.subscribers {
		subscriber.progressCh <- prog
	}
	log.Printf("task %d progress: %d percent %s \n", t.id, percent, desc)
}

func (t *task) done() {
	s := struct{}{}
	for _, subscriber := range t.hub.subscribers {
		subscriber.doneCh <- s
	}
	log.Printf("task %d has done\n", t.id)
}

func (t *task) addSubscriber(s *subscriber) {
	t.hub.subscribers[s.id] = s
	log.Printf("added subscriber %d to task %d\n", s.id, t.id)
}

func (t *task) unsubscribe(id int) {
	close(t.hub.subscribers[id].progressCh)
	close(t.hub.subscribers[id].doneCh)
	delete(t.hub.subscribers, id)
	log.Printf("subscriber %d has been removed from task id %d\n", id, t.id)
}

func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}
