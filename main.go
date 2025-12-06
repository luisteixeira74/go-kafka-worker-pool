package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

type Task struct {
	ID       string `json:"id"`
	Customer string `json:"customer"`
	Payload  string `json:"payload"`
	Msg   kafka.Message
}

type WorkerPool struct {
	numWorkers int
	taskChan   chan Task
	workers    []chan Task
	wg         sync.WaitGroup
	done       chan struct{}
}

func NewWorkerPool(n int) *WorkerPool {
	return &WorkerPool{
		numWorkers: n,
		taskChan:   make(chan Task, 100), // canal global de tasks bufferizado
		workers:    make([]chan Task, n),
		done:       make(chan struct{}),
	}
}

func (wp *WorkerPool) Start(results chan<- kafka.Message) {
	// Inicializa workers
	for i := 0; i < wp.numWorkers; i++ {
		wp.workers[i] = make(chan Task, 10) // canal bufferizado para cada worker
		wp.wg.Add(1)
		go wp.workerLoop(i, wp.workers[i], results)
	}

	// Inicializa dispatcher
	wp.wg.Add(1)
	go wp.dispatcherLoop()
}

func (wp *WorkerPool) dispatcherLoop() {
	defer wp.wg.Done()
	fmt.Println("[dispatcher] iniciado")

	for {
		select {
		case <-wp.done:
			fmt.Println("[dispatcher] encerrado")
			return

		case t := <-wp.taskChan:
			workerID := hash(t.Customer) % wp.numWorkers
			fmt.Printf("[dispatcher] task ID=%s customer=%s -> worker %d\n",
				t.ID, t.Customer, workerID)
			wp.workers[workerID] <- t
		}
	}
}

func (wp *WorkerPool) workerLoop(id int, ch chan Task, results chan<- kafka.Message) {
	defer wp.wg.Done()
	for {
		select {
		case <-wp.done:
			fmt.Printf("[Worker %d] encerrado\n", id)
			return
		case task := <-ch:
			fmt.Printf("[Worker %d] processando task %s (%s)\n",
				id, task.ID, task.Payload)
			time.Sleep(500 * time.Millisecond) // simula processamento

			err := processTask(task)
			if err == nil {
				results <- task.Msg  // commit só em caso de sucesso
			} else {
				log.Printf("[Worker %d] falha processando %s: %v", id, task.ID, err)
				// não enviar para commit, Kafka vai reentregar
			}

		}
	}
}

func processTask(t Task) error {
	return nil
}

func (wp *WorkerPool) Submit(t Task) {
	wp.taskChan <- t
}

func (wp *WorkerPool) Stop() {
	close(wp.done)
	wp.wg.Wait()
}

// -------- CONSUMER KAFKA ----------
func consumerLoop(reader *kafka.Reader, wp *WorkerPool) {
	log.Println("[consumer] iniciado...")

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Println("erro ao ler do kafka:", err)
			continue
		}

		var task Task
		if err := json.Unmarshal(msg.Value, &task); err != nil {
			log.Println("erro ao fazer unmarshal:", err, "value=", string(msg.Value))
			continue
		}

		// Preenche a mensagem original no Task
		task.Msg = msg

		wp.Submit(task)
		log.Printf("[consumer] enviada ao canal -> %+v\n", task)
	}
}

// cria um hash inteiro a partir de uma string
func hash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32())
}

// -------- MAIN --------------------
func main() {
	wp := NewWorkerPool(3) // 3 workers concorrentes
	// canal global para commit de mensagens processadas
    results := make(chan kafka.Message)
	wp.Start(results)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "payment-client-v1",
		GroupID: "payment-consumer-grupo",
	})

    // goroutine que faz commit
    go func() {
        for msg := range results {
            err := reader.CommitMessages(context.Background(), msg)
            if err != nil {
                log.Printf("[commit] falha offset=%d: %v", msg.Offset, err)
            } else {
                log.Printf("[commit] sucesso offset=%d", msg.Offset)
            }
        }
    }()


	go consumerLoop(reader, wp)

	select {} // mantém rodando
}
