# Worker Pool Service

Este serviço implementa um **worker pool** em Go que consome mensagens de um tópico Kafka e processa tasks de forma concorrente.
Cada task possui um campo `customer` no JSON, que determina **qual worker irá processá-la**, garantindo que todas as tasks do mesmo `customer` vão sempre para o mesmo worker.

---

## Entendimento rápido do fluxo da simulação

- O Docker será usado apenas para rodar o Kafka.
- Um terminal será usado para rodar o **kafka-producer** e enviar mensagens.
- Outro terminal (ou mais) rodará o **binário do consumer**, que você irá gerar no item Build.

---

## Modelo de Task

```json
{
  "id": "1",
  "customer": "A",
  "payload": "teste-1"
}
```

### Campos da Task

- **id** – identificador da task
- **customer** – usado para mapear o worker
- **payload** – conteúdo da task

---

## Comportamento do Sistema

O dispatcher lê mensagens do Kafka e calcula:

```
workerID = hash(customer) % numWorkers
```

- Cada worker processa suas tasks **sequencialmente**.
- Tasks com o **mesmo customer** sempre vão para o **mesmo worker**, garantindo ordenação por chave.

---

## Fluxo do Worker Pool

```
Kafka Topic: payment-client-v1
│
▼
[Dispatcher] --- hash(customer) ---> Worker 0
│ ┌--> Worker 1
│ │
│ └--> Worker 2
▼
Task enviada ao worker correspondente
```

O dispatcher recebe as tasks e as envia para o worker baseado no hash do `customer`.
Cada worker processa sequencialmente as tasks que lhe são atribuídas.

---

## Como testar usando CLI do Kafka

### Rodar o Kafka via Docker Compose

```bash
docker compose up -d kafka
```

### Enviar mensagens para o tópico `payment-client-v1`

```bash
docker exec -it kafka kafka-console-producer \
  --broker-list kafka:9092 \
  --topic payment-client-v1 \
  --property "parse.key=false"
```

### Exemplo de mensagens

```json
{"id":"1","customer":"A","payload":"teste-1"}
{"id":"2","customer":"B","payload":"teste-2"}
{"id":"3","customer":"A","payload":"teste-3"}
{"id":"4","customer":"C","payload":"teste-4"}
```

Observe que:

- `customer A` sempre vai para o mesmo worker
- `customer B` para outro
- e assim por diante

---

## Build e execução do binário Go

### Build

```bash
make build
```

### Executar

```bash
make run
```

O binário conecta ao Kafka em `localhost:9092` e inicia o dispatcher + workers.
Os logs indicam qual task foi enviada para qual worker.

### Rodar todos os testes

```bash
make test
```

### Limpar arquivos gerados (Remove artefatos como worker-pool-service e coverage.out)

```bash
make clean
```
