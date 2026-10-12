package websocket

import (
	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
)

func UpgradeMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}

func Handler(hub *Hub) fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		client := &Client{
			hub:  hub,
			conn: c,
			send: make(chan []byte, 256),
		}
		client.hub.register <- client

		go client.writePump()
		client.readPump()
	})
}
