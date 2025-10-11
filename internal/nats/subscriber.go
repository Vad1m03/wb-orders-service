package nats

import (
	"encoding/json"
	"log"
	"wb-orders-service/internal/cache"
	"wb-orders-service/internal/database"
	"wb-orders-service/internal/models"

	"github.com/nats-io/stan.go"
)

type Subscriber struct {
	sc    stan.Conn
	db    *database.PostgresDB
	cache *cache.OrderCache
}

func NewSubscriber(clusterID, clientID, natsURL string, db *database.PostgresDB, cache *cache.OrderCache) (*Subscriber, error) {
	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(natsURL))
	if err != nil {
		return nil, err
	}

	log.Println("✅ Successfully connected to NATS Streaming")
	return &Subscriber{
		sc:    sc,
		db:    db,
		cache: cache,
	}, nil
}

func (s *Subscriber) Subscribe(subject string) error {
	_, err := s.sc.Subscribe(subject, func(m *stan.Msg) {
		log.Printf("📨 Received message on subject %s", subject)
		
		var order models.Order
		if err := json.Unmarshal(m.Data, &order); err != nil {
			log.Printf("❌ Error unmarshaling order: %v", err)
			return
		}

		if order.OrderUID == "" {
			log.Println("❌ Invalid order: missing order_uid")
			return
		}

		if err := s.db.SaveOrder(order); err != nil {
			log.Printf("❌ Error saving order to database: %v", err)
			return
		}

		s.cache.Set(order.OrderUID, order)
		log.Printf("✅ Order %s successfully processed and cached", order.OrderUID)
	}, stan.DurableName("orders-durable"))

	if err != nil {
		return err
	}

	log.Printf("✅ Subscribed to subject: %s", subject)
	return nil
}

func (s *Subscriber) Close() {
	s.sc.Close()
}