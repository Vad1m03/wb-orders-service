package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"wb-orders-service/internal/cache"
	"wb-orders-service/internal/database"
	natsSub "wb-orders-service/internal/nats"

	"github.com/gorilla/mux"
)

var (
	orderCache *cache.OrderCache
	db         *database.PostgresDB
)

func main() {
	log.Println("🚀 Starting WB Orders Service...")

	var err error
	db, err = database.NewPostgresDB("localhost", "5432", "wb_user", "wb_password", "wb_orders")
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	orderCache = cache.NewOrderCache()

	log.Println("📦 Restoring cache from database...")
	orders, err := db.GetAllOrders()
	if err != nil {
		log.Printf("⚠️  Error loading orders from database: %v", err)
	} else {
		orderCache.LoadFromDB(orders)
		log.Printf("✅ Cache restored with %d orders", len(orders))
	}

	subscriber, err := natsSub.NewSubscriber("test-cluster", "order-service", "nats://localhost:4222", db, orderCache)
	if err != nil {
		log.Fatalf("❌ Failed to connect to NATS Streaming: %v", err)
	}
	defer subscriber.Close()

	if err := subscriber.Subscribe("orders"); err != nil {
		log.Fatalf("❌ Failed to subscribe: %v", err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/", homeHandler).Methods("GET")
	router.HandleFunc("/api/order/{id}", getOrderHandler).Methods("GET")

	log.Println("🌐 Starting HTTP server on http://localhost:8080")
	go func() {
		if err := http.ListenAndServe(":8080", router); err != nil {
			log.Fatalf("❌ HTTP server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("👋 Shutting down gracefully...")
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WB Orders Service</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            max-width: 900px;
            margin: 0 auto;
            background: white;
            border-radius: 20px;
            padding: 40px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
        }
        h1 {
            color: #333;
            margin-bottom: 30px;
            text-align: center;
            font-size: 2.5em;
        }
        .search-box {
            display: flex;
            gap: 10px;
            margin-bottom: 30px;
        }
        input {
            flex: 1;
            padding: 15px;
            border: 2px solid #ddd;
            border-radius: 10px;
            font-size: 16px;
            transition: border-color 0.3s;
        }
        input:focus {
            outline: none;
            border-color: #667eea;
        }
        button {
            padding: 15px 30px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 10px;
            font-size: 16px;
            cursor: pointer;
            transition: transform 0.2s;
        }
        button:hover { transform: translateY(-2px); }
        button:active { transform: translateY(0); }
        .result {
            background: #f8f9fa;
            border-radius: 10px;
            padding: 20px;
            margin-top: 20px;
        }
        .error {
            background: #fee;
            color: #c33;
            padding: 15px;
            border-radius: 10px;
            border-left: 4px solid #c33;
        }
        .order-section {
            margin-bottom: 20px;
            background: white;
            padding: 15px;
            border-radius: 8px;
            border-left: 4px solid #667eea;
        }
        .order-section h3 {
            color: #667eea;
            margin-bottom: 10px;
        }
        .field {
            display: flex;
            padding: 8px 0;
            border-bottom: 1px solid #eee;
        }
        .field:last-child { border-bottom: none; }
        .field-name {
            font-weight: bold;
            color: #555;
            min-width: 180px;
        }
        .field-value { color: #333; }
        .item {
            background: #f8f9fa;
            padding: 10px;
            margin: 10px 0;
            border-radius: 5px;
            border-left: 3px solid #764ba2;
        }
        pre {
            background: #2d2d2d;
            color: #f8f8f2;
            padding: 20px;
            border-radius: 8px;
            overflow-x: auto;
            font-size: 14px;
            line-height: 1.5;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🛍️ WB Orders Service</h1>
        <div class="search-box">
            <input type="text" id="orderInput" placeholder="Введите Order UID (например: b563feb7b2b84b6test)">
            <button onclick="searchOrder()">Найти заказ</button>
        </div>
        <div id="result"></div>
    </div>
    <script>
        function searchOrder() {
            const orderUID = document.getElementById('orderInput').value.trim();
            const resultDiv = document.getElementById('result');
            if (!orderUID) {
                resultDiv.innerHTML = '<div class="error">Пожалуйста, введите Order UID</div>';
                return;
            }
            resultDiv.innerHTML = '<p>Загрузка...</p>';
            fetch('/api/order/' + orderUID)
                .then(response => {
                    if (!response.ok) throw new Error('Заказ не найден');
                    return response.json();
                })
                .then(order => {
                    resultDiv.innerHTML = formatOrder(order);
                })
                .catch(error => {
                    resultDiv.innerHTML = '<div class="error">❌ ' + error.message + '</div>';
                });
        }
        function formatOrder(order) {
            return '<div class="result">' +
                '<div class="order-section"><h3>📦 Основная информация</h3>' +
                '<div class="field"><span class="field-name">Order UID:</span><span class="field-value">' + order.order_uid + '</span></div>' +
                '<div class="field"><span class="field-name">Track Number:</span><span class="field-value">' + order.track_number + '</span></div>' +
                '<div class="field"><span class="field-name">Customer ID:</span><span class="field-value">' + order.customer_id + '</span></div></div>' +
                '<div class="order-section"><h3>🚚 Доставка</h3>' +
                '<div class="field"><span class="field-name">Имя:</span><span class="field-value">' + order.delivery.name + '</span></div>' +
                '<div class="field"><span class="field-name">Телефон:</span><span class="field-value">' + order.delivery.phone + '</span></div>' +
                '<div class="field"><span class="field-name">Адрес:</span><span class="field-value">' + order.delivery.address + ', ' + order.delivery.city + '</span></div></div>' +
                '<div class="order-section"><h3>💳 Платёж</h3>' +
                '<div class="field"><span class="field-name">Сумма:</span><span class="field-value">' + order.payment.amount + ' ' + order.payment.currency + '</span></div>' +
                '<div class="field"><span class="field-name">Банк:</span><span class="field-value">' + order.payment.bank + '</span></div></div>' +
                '<div class="order-section"><h3>🛒 Товары</h3>' +
                order.items.map(item => '<div class="item">' +
                    '<div class="field"><span class="field-name">Название:</span><span class="field-value">' + item.name + '</span></div>' +
                    '<div class="field"><span class="field-name">Бренд:</span><span class="field-value">' + item.brand + '</span></div>' +
                    '<div class="field"><span class="field-name">Цена:</span><span class="field-value">' + item.total_price + '</span></div></div>'
                ).join('') + '</div>' +
                '<details style="margin-top: 20px;"><summary style="cursor: pointer; font-weight: bold; padding: 10px; background: #f0f0f0; border-radius: 5px;">📄 JSON</summary>' +
                '<pre>' + JSON.stringify(order, null, 2) + '</pre></details></div>';
        }
        document.getElementById('orderInput').addEventListener('keypress', function(e) {
            if (e.key === 'Enter') searchOrder();
        });
    </script>
</body>
</html>`
	
	t, _ := template.New("index").Parse(tmpl)
	t.Execute(w, nil)
}

func getOrderHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]

	order, exists := orderCache.Get(orderID)
	if !exists {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}