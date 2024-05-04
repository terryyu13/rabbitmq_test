package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	// 創建RabbitMQ連接
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	// 創建RabbitMQ通道
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// 聲明隊列
	q, err := ch.QueueDeclare(
		"hello", // 隊列名稱
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	// 創建Gin路由
	router := gin.Default()

	// 定義發送消息的路由處理函數
	router.GET("/send", func(c *gin.Context) {
		// 發送消息到隊列
		err := ch.Publish(
			"",     // exchange
			q.Name, // routing key
			false,  // mandatory
			false,  // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte("Hello, RabbitMQ!"),
			})
		failOnError(err, "Failed to publish a message")

		c.String(http.StatusOK, "Message sent to RabbitMQ")
	})

	// 定義接收消息的路由處理函數
	router.GET("/receive", func(c *gin.Context) {
		// 創建通道用於接收消息
		msgChan := make(chan []byte)

		// 啟動goroutine接收消息
		go func() {
			// 消費消息
			msgs, err := ch.Consume(
				q.Name, // queue
				"",     // consumer
				true,   // auto-ack
				false,  // exclusive
				false,  // no-local
				false,  // no-wait
				nil,    // args
			)
			failOnError(err, "Failed to register a consumer")

			// 接收消息並發送到通道中
			for d := range msgs {
				msgChan <- d.Body
			}
		}()

		// 從通道中接收消息
		msg := <-msgChan

		// 將消息返回給HTTP客戶端
		c.String(http.StatusOK, "Message received from RabbitMQ: %s", string(msg))
	})

	// 啟動 Web 服務器
	router.Run(":8080")
}
