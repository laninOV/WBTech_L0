package streaming

import (
	"WBTech_L0/internal/database"
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"log"
)

// Streaming представляет собой структуру для обработки данных, полученных через NATS Streaming.
type Streaming struct {
	dbObject *database.DB
}

// NewStream создает новое соединение с NATS Streaming и устанавливает обработчики подписки.
func NewStream(dbInstance *database.DB) (stream *nats.Conn) {
	stream, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Ошибка при подключении к NATS: %v", err)
	}

	NewSubscriber(dbInstance, stream)

	return stream
}

// NewSubscriber устанавливает подписку на канал "intros" в NATS Streaming и связывает обработчик.
func NewSubscriber(dbInstance *database.DB, stream *nats.Conn) (*nats.Subscription, error) {
	subscription, err := stream.Subscribe("intros", func(msg *nats.Msg) {
		SubscribeReceiver(dbInstance, msg)
	})

	if err != nil {
		return nil, err
	}
	return subscription, nil
}

// SubscribeReceiver обрабатывает сообщение, полученное из NATS Streaming, и добавляет информацию о заказе в базу данных.
func SubscribeReceiver(dbInstance *database.DB, msg *nats.Msg) {
	var orderData database.Order

	err := json.Unmarshal([]byte(msg.Data), &orderData)

	if err != nil {
		fmt.Printf("Ошибка при разборе JSON: %v\n", err)
		return
	}

	dbInstance.AddOrderInfo(orderData)

	fmt.Println(orderData.OrderUID)
}
