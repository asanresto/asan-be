package main

import (
	"asan/graph"
	"asan/graph/directives"
	authMiddleware "asan/graph/middleware/auth"
	headerMiddleware "asan/graph/middleware/header"
	"asan/kafka"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/IBM/sarama"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	router := chi.NewRouter()
	router.Use(cors.New(cors.Options{
		// AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedHeaders:   []string{"*"},
		Debug:            true,
		AllowOriginFunc: func(origin string) bool {
			return true
		},
	}).Handler)
	router.Use(headerMiddleware.Middleware())
	router.Use(authMiddleware.Middleware())
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := handler.New(graph.NewExecutableSchema(graph.Config{
		Directives: directives.CustomDirectives,
		Resolvers:  &graph.Resolver{}}))

	srv.AddTransport(&transport.Websocket{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		InitFunc: func(ctx context.Context, initPayload transport.InitPayload) (context.Context, *transport.InitPayload, error) {
			// Don't return non-nil error, it causes an infinite loop in clinet
			jwt := initPayload["Authorization"]
			jwt2, ok := jwt.(string)
			if !ok || jwt2 == "" {
				return ctx, &initPayload, nil
			}
			claims, err := authMiddleware.Parse(strings.Replace(jwt2, "Bearer ", "", 1))
			if err != nil {
				return ctx, &initPayload, nil
			}
			userId, err1 := claims.GetSubject()
			if err1 != nil {
				return ctx, &initPayload, nil
			}
			// For graphql subscription
			ctxNew := context.WithValue(ctx, "userId", userId)
			return ctxNew, &initPayload, nil
		},
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	srv.SetQueryCache(lru.New(1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})
	// Start kafka for chat
	// version, err := sarama.ParseKafkaVersion(sarama.DefaultVersion.String())
	// if err != nil {
	// 	log.Fatalf("Error parsing Kafka version: %v", err)
	// }
	config := sarama.NewConfig()
	// config.Version = version
	// config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRange()}

	consumerGroup, err := sarama.NewConsumerGroup(strings.Split(os.Getenv("KAFKA_PEERS"), ","), "chat", config)
	if err != nil {
		log.Fatalf("Error creating consumer group client: %v", err)
	}

	go func() {
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := consumerGroup.Consume(context.Background(), strings.Split("chat", ","), &kafka.Consumer{}); err != nil {
				if errors.Is(err, sarama.ErrClosedConsumerGroup) {
					return
				}
				log.Fatalf("Error from consumer: %v", err)
			}
		}
	}()

	router.Handle("/", playground.Handler("GraphQL playground", "/query"))
	router.Handle("/query", srv)

	log.Printf("Connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
