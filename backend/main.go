package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof" // Import pprof for profiling

	"github.com/gin-gonic/gin"
	"github.com/mahirjain_10/stock-alert-app/backend/internal/app"
	"github.com/mahirjain_10/stock-alert-app/backend/internal/cron"

	// "github.com/mahirjain_10/stock-alert-app/backend/internal/test"
	"github.com/mahirjain_10/stock-alert-app/backend/internal/types"
	"github.com/mahirjain_10/stock-alert-app/backend/internal/utils"
	"github.com/mahirjain_10/stock-alert-app/backend/internal/websocket"
	"github.com/mahirjain_10/stock-alert-app/backend/web/cmd/router"
)

func main() {

	file ,err := app.InitializeLogger()
	if err != nil{
		slog.Error("message","Error initalizing logger", "error", err)
	}
	defer file.Close()
	// Start pprof server in a separate goroutine
	go func() {
		// log.Println("Starting pprof server on :6060")
		slog.Info("Starting pprof server on :6060")
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Fatalf("pprof server failed: %v", err)
		}
	}()

	// Initialize Gin router
	r := gin.Default()
	err = app.InitalizeEnv()
	if err != nil {
		log.Fatalf("Error loading .env file: %s", err)
		return
	}

	ctx := context.Background()
	// Initialize the database and Redis client using the new helper function
	db, redisClient, err := app.InitializeServices()
	if err != nil {
		log.Fatalf("Error initializing services: %v", err)
		return
	}
	defer db.Close()

	var appInstance = types.App{
		DB:          db,
		RedisClient: redisClient,
	}


	
    // Keep the application running
	// Initialize database tables
	if err := app.InitializeDatabaseTables(db); err != nil {
		log.Fatalf("Error initializing database tables: %v", err)
		return
	}
	
	hub := websocket.NewHub()
	c := cron.StartCron(&appInstance,hub)
	defer c.Stop()
	go hub.Run()

	// Register routes
	go utils.Subscribe(appInstance.RedisClient, ctx)
	go utils.SubscribeToPubSub(appInstance.RedisClient, ctx, "alert-topic")

	
	router.RegisterRoutes(r, hub, &appInstance)
	
	// go func() {
	// 	log.Println("Starting WebSocket Load Test...")
	// 	test.Wstest() // Call load test function
	// }()
	log.Fatal(r.Run(":8080"))
}
