package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq" // Импорт драйвера PostgreSQL
)

// DB представляет собой объект базы данных.
type DB struct {
	name  string
	sqlDb *sql.DB
	csh   *Cache
}

// NewDB создает новый экземпляр DB и устанавливает соединение с базой данных.
func NewDB() (*DB, error) {
	db := DB{}
	db.sqlDb = db.ConnectDB()
	return &db, nil
}

// SendOrderIDToCache добавляет информацию о заказе в кеш базы данных.
func (db *DB) SendOrderIDToCache(oid string) {
	db.sqlDb.QueryContext(context.Background(), `INSERT INTO wb_scheme.cache (order_uid, app_key) VALUES ($1, $2)`, oid, os.Getenv("APP_KEY"))
	log.Printf("%v: OrderID успешно добавлен в кеш (DB)\n", db.name)
}

// ClearCache очищает кеш базы данных.
func (db *DB) ClearCache() {
	_, err := db.sqlDb.ExecContext(context.Background(), `DELETE FROM cache WHERE app_key = $1`, os.Getenv("APP_KEY"))
	if err != nil {
		log.Printf("%v: ошибка очистки кеша: %s\n", db.name, err)
	}
	log.Printf("%v: кеш успешно очищен из базы данных\n", db.name)
}

// SetCacheInstance устанавливает объект кеша для базы данных.
func (db *DB) SetCacheInstance(csh *Cache) {
	db.csh = csh
}

// GetCacheState получает состояние кеша.
func (db *DB) GetCacheState(bufSize int) (map[string]interface{}, []string, int, error) {
	buffer := make(map[string]interface{}, bufSize)
	queue := make([]string, bufSize)
	var queueInd int

	query := fmt.Sprintf("SELECT wb_scheme.cache.order_uid FROM wb_scheme.cache WHERE app_key = '%s' ORDER BY id DESC LIMIT %d", os.Getenv("APP_KEY"), bufSize)
	rows, err := db.sqlDb.QueryContext(context.Background(), query)
	if err != nil {
		log.Printf("%v: не удалось получить order_uid из базы данных: %v\n", db.name, err)
	}
	defer rows.Close()

	var oid string
	for rows.Next() {
		if err := rows.Scan(&oid); err != nil {
			log.Printf("%v: не удалось получить oid из строки базы данных: %v\n", db.name, err)
			return buffer, queue, queueInd, errors.New("не удалось получить oid из строки базы данных")
		}
		queue[queueInd] = oid
		queueInd++

		data, exists := db.csh.Get(oid)
		if exists {
			buffer[oid] = data
		}
	}

	if queueInd == 0 {
		return buffer, queue, queueInd, errors.New("кеш пуст")
	}

	// Разворачиваем очередь, чтобы получить новые данные в начале.
	for i := 0; i < int(queueInd/2); i++ {
		queue[i], queue[queueInd-i-1] = queue[queueInd-i-1], queue[i]
	}

	return buffer, queue, queueInd, nil
}

// AddOrderInfo добавляет информацию о заказе в базу данных.
func (db *DB) AddOrderInfo(orderData Order) (int64, error) {
	// Начинаем транзакцию для выполнения нескольких SQL-запросов.
	tx, err := db.sqlDb.BeginTx(context.Background(), nil)
	if err != nil {
		log.Println("Невозможно начать транзакцию", err)
		return 0, err
	}
	defer tx.Rollback()

	var lastInsertPaymentID int64
	var lastInsertDeliveryID int64
	var lastInsertItemID int64
	var orderItemsIds []int64 = []int64{}
	var lastOrderItemID string

	// SQL-запросы для вставки информации о платеже, доставке, товарах и заказе.
	// Все данные вставляются внутри транзакции, чтобы обеспечить атомарность операций.
	stmtPayment := `
		INSERT INTO wb_scheme.payment (transaction, request_id, currency, provider, amount, payment_dt, bank, 
			delivery_cost, goods_total, custom_fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id
	`

	err = tx.QueryRowContext(context.Background(), stmtPayment, orderData.Payment.Transaction, orderData.Payment.RequestId, orderData.Payment.Currency, orderData.Payment.Provider, orderData.Payment.Amount,
		orderData.Payment.PaymentDt, orderData.Payment.Bank, orderData.Payment.DeliveryCost, orderData.Payment.GoodsTotal, orderData.Payment.CustomFee).Scan(&lastInsertPaymentID)

	if err != nil {
		fmt.Printf("Ошибка вставки данных о платеже: %v\n", err)
		return 0, err
	}

	stmtDelivery := `
		INSERT INTO wb_scheme.delivery (name, phone, zip, city, address, region, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id
	`

	err = tx.QueryRowContext(context.Background(), stmtDelivery, orderData.Delivery.Name, orderData.Delivery.Phone, orderData.Delivery.Zip, orderData.Delivery.City, orderData.Delivery.Address, orderData.Delivery.Region, orderData.Delivery.Email).Scan(&lastInsertDeliveryID)

	if err != nil {
		fmt.Printf("Ошибка вставки данных о доставке: %v\n", err)
		return 0, err
	}

	stmtItem := `
		INSERT INTO wb_scheme.items (chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING item_id
	`

	for _, item := range orderData.Items {

		err = tx.QueryRowContext(context.Background(), stmtItem, item.ChrtID, item.TrackNumber, item.Price, item.RID, item.Name, item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status).Scan(&lastInsertItemID)

		if err != nil {
			fmt.Printf("Ошибка вставки данных о товаре: %v\n", err)
			return 0, err
		}

		orderItemsIds = append(orderItemsIds, lastInsertItemID)
	}

	stmtOrder := `
		INSERT INTO wb_scheme.orders (order_uid, payment_id, delivery_id, track_number, entry, locale, internal_signature, delivery_service, shardkey, sm_id, date_created, oof_shard, customer_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING order_uid
	`

	err = tx.QueryRowContext(context.Background(), stmtOrder, orderData.OrderUID, lastInsertPaymentID, lastInsertDeliveryID, orderData.TrackNumber, orderData.Entry, orderData.Locale, orderData.InternalSignature, orderData.DeliveryService, orderData.Shardkey, orderData.SMID, orderData.DateCreated, orderData.OofShard, orderData.CustomerID).Scan(&lastOrderItemID)

	if err != nil {
		fmt.Printf("Ошибка вставки данных о заказе: %v\n", err)
		return 0, err
	}

	stmtOrderItems := `
		INSERT INTO wb_scheme.order_items (order_uid, item_id)
		VALUES ($1, $2)
	`

	for _, itemId := range orderItemsIds {

		_, err := tx.ExecContext(context.Background(), stmtOrderItems, lastOrderItemID, itemId)

		if err != nil {
			log.Printf("Не удалось вставить данные (order_items)")
			return 0, err
		}
	}

	// Если все успешно, фиксируем транзакцию.
	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	log.Println("Заказ успешно добавлен в базу данных")

	return 0, nil
}

// GetOrderByUid получает информацию о заказе по его уникальному идентификатору.
func (db *DB) GetOrderByUid(orderUid string) (Order, error) {
	var order Order

	stmt := `
	select wb_scheme.orders.order_uid, wb_scheme.orders.track_number, wb_scheme.orders.entry,
	wb_scheme.orders.locale, wb_scheme.orders.internal_signature, wb_scheme.orders.delivery_service,
	wb_scheme.orders.shardkey, wb_scheme.orders.sm_id, wb_scheme.orders.oof_shard, wb_scheme.orders.date_created,

	wb_scheme.delivery.name, wb_scheme.delivery.phone, wb_scheme.delivery.zip, wb_scheme.delivery.city,
	wb_scheme.delivery.address, wb_scheme.delivery.region, wb_scheme.delivery.email,

	wb_scheme.payment.transaction, wb_scheme.payment.request_id, wb_scheme.payment.currency,
	wb_scheme.payment.provider, wb_scheme.payment.amount, wb_scheme.payment.payment_dt,
	wb_scheme.payment.bank, wb_scheme.payment.delivery_cost, wb_scheme.payment.goods_total,
	wb_scheme.payment.custom_fee

	from wb_scheme.orders

	inner join wb_scheme.delivery on wb_scheme.delivery.id = wb_scheme.orders.delivery_id

	inner join wb_scheme.payment on wb_scheme.payment.id = wb_scheme.orders.payment_id

	where wb_scheme.orders.order_uid = $1
	`

	err := db.sqlDb.QueryRowContext(context.Background(), stmt, orderUid).Scan(
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature, &order.DeliveryService,
		&order.Shardkey, &order.SMID, &order.OofShard, &order.DateCreated,

		&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip, &order.Delivery.City, &order.Delivery.Address,
		&order.Delivery.Region, &order.Delivery.Email,

		&order.Payment.Transaction, &order.Payment.RequestId, &order.Payment.Currency, &order.Payment.Provider,
		&order.Payment.Amount, &order.Payment.PaymentDt, &order.Payment.Bank, &order.Payment.DeliveryCost,
		&order.Payment.GoodsTotal, &order.Payment.CustomFee)

	if err != nil {
		fmt.Print("idd", err)
		return order, errors.New("не удалось получить заказ из базы данных")
	}

	stmtItems := `
	select wb_scheme.order_items.item_id from wb_scheme.order_items where wb_scheme.order_items.order_uid = $1
	`
	rowsItems, err := db.sqlDb.Query(stmtItems, orderUid)
	if err != nil {
		fmt.Print("error", err)
		return order, errors.New("не удалось получить список идентификаторов товаров из базы данных")
	}

	stmtItem := `
	select wb_scheme.items.chrt_id, wb_scheme.items.track_number, wb_scheme.items.price, wb_scheme.items.rid,
	wb_scheme.items.name, wb_scheme.items.sale, wb_scheme.items.size, wb_scheme.items.total_price,
	wb_scheme.items.nm_id, wb_scheme.items.brand, wb_scheme.items.status
	from wb_scheme.items where wb_scheme.items.item_id = $1
	`

	var itemID int64
	for rowsItems.Next() {
		var item Item
		if err := rowsItems.Scan(&itemID); err != nil {
			return order, errors.New("не удалось получить идентификатор товара из строки базы данных")
		}

		err = db.sqlDb.QueryRowContext(context.Background(), stmtItem, itemID).Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.RID,
			&item.Name, &item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status)
		if err != nil {
			return order, errors.New("не удалось получить товар из базы данных")
		}
		order.Items = append(order.Items, item)
	}

	return order, nil
}

// RunExecCommand выполняет SQL-команду на базе данных.
func RunExecCommand(dbInstance *sql.DB, command string) {
	_, err := dbInstance.Exec(command)
	if err != nil {
		panic(err)
	}
}
