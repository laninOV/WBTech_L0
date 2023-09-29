package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/ddosify/go-faker/faker"
	"github.com/nats-io/nats.go"
)

// Delivery представляет информацию о доставке.
type Delivery struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

// Payment представляет информацию о платеже.
type Payment struct {
	Transaction  string `json:"transaction"`
	RequestId    string `json:"request_id"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int    `json:"amount"`
	PaymentDt    string `json:"payment_dt"`
	Bank         string `json:"bank"`
	DeliveryCost int    `json:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total"`
	CustomFee    int    `json:"custom_fee"`
}

// Item представляет информацию о товаре.
type Item struct {
	ChrtID      int    `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int    `json:"price"`
	RID         string `json:"rid"`
	Name        string `json:"name"`
	Sale        int    `json:"sale"`
	Size        int    `json:"size"`
	TotalPrice  int    `json:"total_price"`
	NmID        int    `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int    `json:"status"`
}

// Order представляет информацию о заказе.
type Order struct {
	OrderUID          string   `json:"order_uid"`
	TrackNumber       string   `json:"track_number"`
	Entry             string   `json:"entry"`
	Delivery          Delivery `json:"delivery"`
	Payment           Payment  `json:"payment"`
	Items             []Item   `json:"items"`
	Locale            string   `json:"locale"`
	InternalSignature string   `json:"internal_signature"`
	CustomerID        int      `json:"customer_id"`
	DeliveryService   string   `json:"delivery_service"`
	Shardkey          int      `json:"shardkey"`
	SMID              int      `json:"sm_id"`
	DateCreated       string   `json:"date_created"`
	OofShard          int      `json:"oof_shard"`
}

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("can't connect to NATS: %v", err)
	}
	defer nc.Close()

	// Публикация в канал
	count := 0
	for {
		randomJSON := GenerateRandomJSONData()
		nc.Publish("intros", []byte(randomJSON))
		count++
		log.Printf("sent JSON %v", count)
		time.Sleep(5 * time.Second)
	}
}

// GenerateRandomJSONData генерирует случайные данные заказа в формате JSON.
func GenerateRandomJSONData() string {
	faker := faker.NewFaker()

	order := Order{
		OrderUID:    faker.RandomUUID().String(),
		TrackNumber: strings.ToUpper("WB" + faker.RandomLoremWord()),
		Entry:       strings.ToUpper("WBIL" + faker.RandomLoremWord()),
		Delivery: Delivery{
			Name:    faker.RandomPersonFirstName(),
			Phone:   faker.RandomPhoneNumberExt(),
			Zip:     faker.RandomBankAccount(),
			City:    faker.RandomAddressCity(),
			Address: faker.RandomAddressStreetAddress(),
			Region:  faker.RandomAddressCountry(),
			Email:   faker.RandomEmail(),
		},
		Payment: Payment{
			Transaction:  faker.RandomPassword(),
			RequestId:    faker.RandomBankAccount(),
			Currency:     faker.RandomCurrencyCode(),
			Provider:     faker.RandomCompanyName(),
			Amount:       faker.RandomIntBetween(100, 10000),
			PaymentDt:    faker.RandomBankAccount(),
			Bank:         faker.RandomCompanyName(),
			DeliveryCost: faker.RandomIntBetween(100, 10000),
			GoodsTotal:   faker.RandomIntBetween(100, 10000),
			CustomFee:    faker.RandomIntBetween(100, 10000),
		},
		Items: []Item{
			{
				ChrtID:      faker.RandomIntBetween(100000, 9999999),
				TrackNumber: faker.RandomBankAccountIban(),
				Price:       faker.RandomIntBetween(100, 10000),
				RID:         faker.RandomBankAccountIban(),
				Name:        faker.RandomProductName(),
				Sale:        faker.RandomIntBetween(5, 99),
				Size:        faker.RandomIntBetween(0, 10),
				TotalPrice:  faker.RandomIntBetween(100, 10000),
				NmID:        faker.RandomIntBetween(1000, 99999),
				Brand:       faker.RandomCompanyName(),
				Status:      faker.RandomIntBetween(100, 600),
			},
		},
		Locale:            faker.RandomLocale(),
		InternalSignature: faker.RandomUUID().String(),
		CustomerID:        faker.IntBetween(0, 100),
		DeliveryService:   faker.RandomCompanyName(),
		Shardkey:          faker.IntBetween(0, 10),
		SMID:              faker.IntBetween(10, 100),
		DateCreated:       faker.RandomDatePast(),
		OofShard:          faker.IntBetween(0, 10),
	}

	jsonData, err := json.MarshalIndent(order, "", " ")
	if err != nil {
		fmt.Println("Error:", err)
	}

	return string(jsonData)
}
