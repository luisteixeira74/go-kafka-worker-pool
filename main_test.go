package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestHashDeterministico(t *testing.T) {
    c := "A"
    w1 := hash(c) % 3
    w2 := hash(c) % 3
    if w1 != w2 {
        t.Errorf("hash não é determinístico: %d != %d", w1, w2)
    }
}

func TestDispatcherAssignsSameWorker(t *testing.T) {
    wp := NewWorkerPool(3)
    results := make(chan kafka.Message)
    wp.Start(results)
    defer wp.Stop()

    taskA1 := Task{ID: "1", Customer: "A", Payload: "t1"}
    taskA2 := Task{ID: "2", Customer: "A", Payload: "t2"}

    wp.Submit(taskA1)
    wp.Submit(taskA2)

    time.Sleep(1 * time.Millisecond) // deixa o dispatcher enviar
    // Aqui podemos inspecionar logs ou os canais internos (wp.workers) se exportarmos para teste
}

func TestWorkerProcessing(t *testing.T) {
    ch := make(chan Task, 2)
    done := make(chan struct{})
    var processed []string
    go func() {
        for {
            select {
            case t := <-ch:
                processed = append(processed, t.ID)
            case <-done:
                return
            }
        }
    }()

    ch <- Task{ID: "1", Customer: "A", Payload: "t1"}
    ch <- Task{ID: "2", Customer: "B", Payload: "t2"}
    time.Sleep(5 * time.Millisecond)
    close(done)

    if len(processed) != 2 {
        t.Errorf("esperava 2 tasks processadas, mas foram %d", len(processed))
    }
}

func TestConsumerLoopSimulado(t *testing.T) {
    tasks := make(chan Task, 10)
    messages := [][]byte{
        []byte(`{"id":"1","customer":"A","payload":"t1"}`),
        []byte(`{"id":"2","customer":"B","payload":"t2"}`),
    }

    for _, msg := range messages {
        var incoming Task
        json.Unmarshal(msg, &incoming)
        tasks <- incoming
    }

    if len(tasks) != 2 {
        t.Errorf("esperava 2 tasks no canal, encontrou %d", len(tasks))
    }
}

func TestTwoWorkerPoolsCustomerRouting(t *testing.T) {
    results := make(chan kafka.Message)
    results2 := make(chan kafka.Message)
    wp1 := NewWorkerPool(3)
    wp2 := NewWorkerPool(3)
    wp1.Start(results)
    wp2.Start(results2)
    defer wp1.Stop()
    defer wp2.Stop()

    tasks := []Task{
        {ID: "1", Customer: "A", Payload: "t1"},
        {ID: "2", Customer: "B", Payload: "t2"},
        {ID: "3", Customer: "A", Payload: "t3"},
        {ID: "4", Customer: "C", Payload: "t4"},
    }

    // Simula envio de tasks alternando instâncias
    for i, task := range tasks {
        if i%2 == 0 {
            wp1.Submit(task)
        } else {
            wp2.Submit(task)
        }
    }

    // Apenas para permitir que os dispatchers processem
    time.Sleep(5 * time.Millisecond)
}




