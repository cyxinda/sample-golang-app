package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"

	"github.com/SigNoz/sample-golang-app/controllers"
	"github.com/SigNoz/sample-golang-app/metrics"
	"github.com/SigNoz/sample-golang-app/models"
	"github.com/gin-gonic/gin"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"

	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	serviceName  = os.Getenv("SERVICE_NAME")
	collectorURL = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	insecure     = os.Getenv("INSECURE_MODE")
)

func initLog() func(context.Context) error {
	ctx := context.Background()
	var secureOption otlploghttp.Option
	if strings.ToLower(insecure) == "false" || insecure == "0" || strings.ToLower(insecure) == "f" {
		// secureOption = otlptracehttp.WithTLSClientConfig(credentials.NewTLS()(nil, ""))
	} else {
		secureOption = otlploghttp.WithInsecure()
	}
	logExporter, err := otlploghttp.New(ctx, otlploghttp.WithEndpoint(collectorURL), secureOption)
	if err != nil {
		panic("failed to initialize exporter")
	}
	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("library.language", "go"),
		),
	)
	// Create the logger provider
	logProvider := log.NewLoggerProvider(
		log.WithProcessor(
			log.NewBatchProcessor(logExporter),
		),
		log.WithResource(resources),
	)

	// Ensure the logger is shutdown before exiting so all pending logs are exported
	// defer logProvider.Shutdown(ctx)
	// Set the logger provider globally
	global.SetLoggerProvider(logProvider)

	// Instantiate a new slog logger
	// logger := otelslog.NewLogger("goApp")
	// You can use the logger directly anywhere in your app now
	slog.SetDefault(otelslog.NewLogger(serviceName, otelslog.WithLoggerProvider(logProvider)))
	// logger.Debug("Something interesting happened cccccccccccccccccccccccccc")
	// Logger := otelslog.NewLogger("my/pkg/name", otelslog.WithLoggerProvider(logProvider))
	// // Logger.InfoContext(ctx, "shutdown service")
	// Logger.Info("cccccccccccccccccccccccccccccccccccc")
	slog.Info("ccccccccccccccccccccccccccccckkkkkkkkkkkkkkkkkkkkkkkkkkkkkk..............................................................")
	return logExporter.Shutdown
}

func initTracer() func(context.Context) error {

	var secureOption otlptracehttp.Option
	fmt.Println("INSECURE_MODE:::::::" + insecure)
	if strings.ToLower(insecure) == "false" || insecure == "0" || strings.ToLower(insecure) == "f" {
		// secureOption = otlptracehttp.WithTLSClientConfig(credentials.NewTLS()(nil, ""))
	} else {
		secureOption = otlptracehttp.WithInsecure()
	}

	exporter, err := otlptrace.New(
		context.Background(),
		otlptracehttp.NewClient(
			secureOption,
			otlptracehttp.WithEndpoint(collectorURL),
		),
	)

	if err != nil {
		slog.Error("Failed to create exporter: %v", err)
	}
	resources, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("library.language", "go"),
		),
	)
	if err != nil {
		slog.Error("Could not set resources: %v", err)
	}

	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(resources),
		),
	)
	return exporter.Shutdown
}

func main() {

	// // 打开或创建日志文件
	// logfile, err := os.OpenFile("/data/logs/test.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	// if err != nil {
	// 	log.Fatalf("无法打开日志文件: %v", err)
	// }
	// // 延迟关闭文件
	// defer logfile.Close()

	// // 设置日志输出到文件
	// log.SetOutput(logfile)

	// // 可选：设置日志前缀和标志
	// log.SetPrefix("MyApp: ")
	// log.SetFlags(log.LstdFlags)

	// // 测试日志输出
	// log.Println("这是一个测试日志消息")
	// Create the OTLP log exporter that sends logs to configured destination
	cleanupLog := initLog()
	defer cleanupLog(context.Background())
	// You can use the logger directly anywhere in your app now
	cleanup := initTracer()
	defer cleanup(context.Background())

	provider := metrics.InitMeter()
	defer provider.Shutdown(context.Background())

	meter := provider.Meter("sample-golang-app")
	metrics.GenerateMetrics(meter)

	r := gin.Default()
	r.Use(otelgin.Middleware(serviceName))
	// Connect to database
	models.ConnectDatabase()

	// Routes
	r.GET("/books", controllers.FindBooks)
	r.GET("/books/:id", controllers.FindBook)
	r.POST("/books", controllers.CreateBook)
	r.PATCH("/books/:id", controllers.UpdateBook)
	r.DELETE("/books/:id", controllers.DeleteBook)

	// Run the server
	r.Run(":8090")
}
