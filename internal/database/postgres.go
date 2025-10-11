package database

import (
	"database/sql"
	"fmt"
	"log"
	"wb-orders-service/internal/models"

	_ "github.com/lib/pq"
)

type PostgresDB struct {
	conn *sql.DB
}

func NewPostgresDB(host, port, user, password, dbname string) (*PostgresDB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	log.Println("✅ Successfully connected to PostgreSQL")
	return &PostgresDB{conn: db}, nil
}

func (db *PostgresDB) SaveOrder(order models.Order) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, 
						   customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (order_uid) DO NOTHING`,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard)
	
	if err != nil {
		return fmt.Errorf("error inserting order: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO deliveries (order_uid, name, phone, zip, city, address, region, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	
	if err != nil {
		return fmt.Errorf("error inserting delivery: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO payments (order_uid, transaction, request_id, currency, provider, 
							 amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
		order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
	
	if err != nil {
		return fmt.Errorf("error inserting payment: %w", err)
	}

	for _, item := range order.Items {
		_, err = tx.Exec(`
			INSERT INTO items (order_uid, chrt_id, track_number, price, rid, name, 
							  sale, size, total_price, nm_id, brand, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name,
			item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		
		if err != nil {
			return fmt.Errorf("error inserting item: %w", err)
		}
	}

	return tx.Commit()
}

func (db *PostgresDB) GetAllOrders() ([]models.Order, error) {
	rows, err := db.conn.Query(`SELECT order_uid, track_number, entry, locale, internal_signature,
									   customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
								FROM orders`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale,
			&order.InternalSignature, &order.CustomerID, &order.DeliveryService,
			&order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard)
		if err != nil {
			log.Printf("Error scanning order: %v", err)
			continue
		}

		err = db.conn.QueryRow(`SELECT name, phone, zip, city, address, region, email 
								FROM deliveries WHERE order_uid = $1`, order.OrderUID).
			Scan(&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
				&order.Delivery.City, &order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email)
		if err != nil {
			log.Printf("Error loading delivery for order %s: %v", order.OrderUID, err)
		}

		err = db.conn.QueryRow(`SELECT transaction, request_id, currency, provider, amount,
									   payment_dt, bank, delivery_cost, goods_total, custom_fee
								FROM payments WHERE order_uid = $1`, order.OrderUID).
			Scan(&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency,
				&order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDt,
				&order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal, &order.Payment.CustomFee)
		if err != nil {
			log.Printf("Error loading payment for order %s: %v", order.OrderUID, err)
		}

		itemRows, err := db.conn.Query(`SELECT chrt_id, track_number, price, rid, name, sale,
											   size, total_price, nm_id, brand, status
										FROM items WHERE order_uid = $1`, order.OrderUID)
		if err != nil {
			log.Printf("Error loading items for order %s: %v", order.OrderUID, err)
		} else {
			defer itemRows.Close()
			for itemRows.Next() {
				var item models.Item
				err := itemRows.Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid,
					&item.Name, &item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status)
				if err != nil {
					log.Printf("Error scanning item: %v", err)
					continue
				}
				order.Items = append(order.Items, item)
			}
		}

		orders = append(orders, order)
	}

	return orders, nil
}

func (db *PostgresDB) Close() error {
	return db.conn.Close()
}