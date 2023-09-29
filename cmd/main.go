package main

import (
	// Импортируем необходимые пакеты
	"WBTech_L0/cmd/configuration"
	"WBTech_L0/internal/database"
	"WBTech_L0/internal/streaming"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	// Импортируем пакеты для инициализации
	_ "WBTech_L0/cmd/configuration"
	_ "WBTech_L0/internal/database"
	_ "WBTech_L0/internal/streaming"

	"github.com/gorilla/mux"
)

func main() {
	// Выполняем настройку конфигурации приложения
	configuration.ConfigSetup()

	// Создаем экземпляр базы данных
	dbInstance, err := database.NewDB()
	if err == nil {
		fmt.Println("База данных подключена!")
	}

	// Создаем экземпляр кэша
	csh := database.NewCache(dbInstance)

	// Инициализируем потоковую обработку данных
	streaming.NewStream(dbInstance)

	// Создаем маршрутизатор для обработки HTTP-запросов
	r := mux.NewRouter()

	// Определяем обработчики для API-маршрутов
	r.HandleFunc("/api/getOrderInfo", GettingOrderInfoByOrderUID).Methods("GET")
	r.HandleFunc("/api/getOrderInfo/{orderUID}", func(w http.ResponseWriter, r *http.Request) {
		GettingOrderInfo(w, r, csh)
	}).Methods("GET")

	// Настроим обработку корневого URL
	http.Handle("/", r)

	fmt.Println("Сервер работает на порту :8080...")
	http.ListenAndServe(":8080", nil)
}

// GettingOrderInfoByOrderUID обрабатывает запрос для получения информации о заказе по его уникальному идентификатору (OrderUID).
func GettingOrderInfoByOrderUID(w http.ResponseWriter, r *http.Request) {
	// Парсим HTML-шаблон
	tmpl, err := template.ParseFiles("cmd/template/template.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Выполняем шаблонизацию и отправляем HTML-страницу клиенту
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// GettingOrderInfo обрабатывает запрос для получения информации о заказе по его уникальному идентификатору (OrderUID).
func GettingOrderInfo(w http.ResponseWriter, r *http.Request, dbInstance *database.Cache) {
	// Устанавливаем заголовок Content-Type для ответа
	w.Header().Set("Content-Type", "application/json")

	// Извлекаем параметры из URL
	vars := mux.Vars(r)
	orderUID := vars["orderUID"]

	// Получаем информацию о заказе из кэша
	orderFetch, err := dbInstance.DBInst.GetOrderByUid(orderUID)

	if err != nil {
		// В случае ошибки возвращаем статус "500 Internal Server Error"
		http.Error(w, "Не удалось получить информацию о заказе из базы данных", http.StatusInternalServerError)
		return
	}

	// Создаем структуру Order для ответа
	order := database.Order{
		OrderUID:          orderFetch.OrderUID,
		TrackNumber:       orderFetch.TrackNumber,
		Entry:             orderFetch.Entry,
		Delivery:          orderFetch.Delivery,
		Payment:           orderFetch.Payment,
		Items:             orderFetch.Items,
		Locale:            orderFetch.Locale,
		InternalSignature: orderFetch.InternalSignature,
		CustomerID:        orderFetch.CustomerID,
		DeliveryService:   orderFetch.DeliveryService,
		Shardkey:          orderFetch.Shardkey,
		SMID:              orderFetch.SMID,
		DateCreated:       orderFetch.DateCreated,
		OofShard:          orderFetch.OofShard,
	}

	// Кодируем структуру в JSON и отправляем клиенту
	json.NewEncoder(w).Encode(order)
}
