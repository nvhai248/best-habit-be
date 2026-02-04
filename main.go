package main

import (
	"bestHabit/component"
	"bestHabit/component/cronjob"
	"bestHabit/component/mailprovider"
	"bestHabit/component/oauthprovider"
	"bestHabit/component/sendnotificationprovider"
	"bestHabit/component/uploadprovider"
	"bestHabit/docs"
	"bestHabit/middleware"
	"bestHabit/modules/challenge/challengetransport/ginchallenge"
	"bestHabit/modules/habit/habittransport/ginhabit"
	"bestHabit/modules/participant/participanttransport/ginparticipant"
	"bestHabit/modules/statistical/statisticaltransport/ginstatistical"
	"bestHabit/modules/task/tasktransport/gintask"
	"bestHabit/modules/user/usertransport/ginuser"
	"bestHabit/pubsub/pblocal"
	"bestHabit/skio"
	"bestHabit/subscriber"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"strconv"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func ConnectToDB(dns string) *sqlx.DB {
	db, err := sqlx.Connect("mysql", dns)
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}
	// Check connection to DB by Ping
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping the database:", err)
	}
	fmt.Println("Connected to the database!")
	return db
}

func runServer(db *sqlx.DB,
	secretKey string,
	s3upProvider uploadprovider.UploadProvider,
	gmailSender mailprovider.EmailSender,
	authProvider oauthprovider.GGOAuthProvider,
	cronProvider cronjob.CronJobProvider,
	sendNotificationProvider sendnotificationprovider.NotificationProvider,
) {

	appCtx := component.NewAppContext(db, secretKey,
		s3upProvider,
		pblocal.NewPubSub(),
		gmailSender,
		authProvider,
		cronProvider,
		sendNotificationProvider)

	router := gin.Default()

	// 1. Security Headers
	router.Use(middleware.SecurityHeaders())

	// 2. Rate Limiting
	rpsStr := os.Getenv("RATE_LIMIT_RPS")
	burstStr := os.Getenv("RATE_LIMIT_BURST")
	rps, _ := strconv.ParseFloat(rpsStr, 64)
	burst, _ := strconv.Atoi(burstStr)
	if rps <= 0 {
		rps = 10 // default
	}
	if burst <= 0 {
		burst = 20 // default
	}
	router.Use(middleware.RateLimit(rps, burst))

	// 3. CORS
	allowedOrigins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
	if len(allowedOrigins) == 0 || allowedOrigins[0] == "" {
		allowedOrigins = []string{"*"}
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Accept", "Accept-Language", "Authorization", "Content-Type", "X-Requested-With", "Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 3600,
	}))

	router.Use(middleware.Recover(appCtx))

	// start subscriber
	if err := subscriber.NewEngine(appCtx).Start(); err != nil {
		log.Fatal(err)
	}

	rtEngine := skio.NewEngine()

	if err := rtEngine.Run(appCtx, router); err != nil {
		log.Fatalln("Failed to start server: ", err)
	}

	docs.SwaggerInfo.BasePath = "/api"

	routerAPIS := router.Group("/api")

	log_and_register := routerAPIS.Group("/")
	{
		log_and_register.GET("/ping", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{
				"message": "Ping OK!!!!",
			})
		})
		log_and_register.GET("/mong1", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{
				"message": "Happy new year 2024, Wish everything perfectly good!",
			})
		})
		log_and_register.GET("/mong2", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{
				"message": "Happy new year 2024, wish in new year had a new job!",
			})
		})
		log_and_register.GET("/mong3", func(ctx *gin.Context) {
			ctx.JSON(http.StatusOK, gin.H{
				"message": "Happy new year 2024, wish in new year had a new job and had a perfect love with girlfriend!",
			})
		})
		log_and_register.POST("/register", ginuser.BasicRegister(appCtx))
		log_and_register.POST("/login", ginuser.BasicLogin(appCtx))
		log_and_register.POST("/users/send-reset-password", ginuser.SenResetPw(appCtx))
		// google oauth
		log_and_register.GET("/auth/google", ginuser.HandleGoogleLogin(appCtx))
		log_and_register.GET("/auth/google/callback", ginuser.HandleGoogleCallback(appCtx))
	}

	ApisUser(appCtx, routerAPIS, db)

	task := routerAPIS.Group("/tasks", middleware.RequireAuth(appCtx), middleware.IsVerifiedUser(appCtx))
	{
		task.POST("/", gintask.CreateTask(appCtx))
		task.GET("/", gintask.ListTaskByConditions(appCtx))
		task.GET("/:id", gintask.FindTask(appCtx))
		task.PATCH("/:id", gintask.UpdateTask(appCtx))
		task.DELETE("/:id", gintask.SoftDeleteTask(appCtx))
	}

	habit := routerAPIS.Group("/habits", middleware.RequireAuth(appCtx), middleware.IsVerifiedUser(appCtx))
	{
		habit.POST("/", ginhabit.CreateHabit(appCtx))
		habit.GET("/", ginhabit.ListHabitByConditions(appCtx))
		habit.GET("/:id", ginhabit.FindHabit(appCtx))
		habit.PATCH("/:id", ginhabit.UpdateHabit(appCtx))
		habit.DELETE("/:id", ginhabit.SoftDeleteHabit(appCtx))
		habit.PATCH("/:id/confirm-completed", ginhabit.AddCompletedDate(appCtx))
	}

	challenge := routerAPIS.Group("/challenges", middleware.RequireAuth(appCtx), middleware.IsVerifiedUser(appCtx))
	{
		challenge.GET("/", ginchallenge.ListChallengeByConditions(appCtx))
		challenge.GET("/:id", ginchallenge.FindChallenge(appCtx))

		challenge.POST("/:id/user-join", ginparticipant.CreateParticipant(appCtx))
		challenge.DELETE("/:id/user-cancel", ginparticipant.CancelParticipant(appCtx))
		challenge.GET("/participants", ginparticipant.ListChallengeJoined(appCtx))
		challenge.PATCH("/participants/:id", ginparticipant.UpdateParticipant(appCtx))
		challenge.GET("/participants/:id", ginparticipant.FindParticipant(appCtx))
	}

	challengeAdmin := routerAPIS.Group("/challenges", middleware.RequireAuth(appCtx), middleware.RequireRoles(appCtx, "admin"))
	{
		challengeAdmin.POST("/", ginchallenge.CreateChallenge(appCtx))
		challengeAdmin.PATCH("/:id", ginchallenge.UpdateChallenge(appCtx))
		challengeAdmin.DELETE("/:id", ginchallenge.DeleteChallenge(appCtx))
	}

	userAdmin := routerAPIS.Group("/users", middleware.RequireAuth(appCtx), middleware.RequireRoles(appCtx, "admin"))
	{
		userAdmin.GET("/", ginuser.ListUserByConditions(appCtx))
		userAdmin.GET("/:id", ginuser.FindUser(appCtx))
		userAdmin.PATCH("/:id/banned", ginuser.BannedUser(appCtx))
		userAdmin.PATCH("/:id/unbanned", ginuser.UnbannedUser(appCtx))
		userAdmin.DELETE("/:id", ginuser.DeleteUser(appCtx))
	}

	routerAPIS.GET("/statistical",
		middleware.RequireAuth(appCtx),
		middleware.RequireRoles(appCtx, "admin"),
		ginstatistical.GetStatistical(appCtx))

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.Run(":8080")
}

func main() {
	// Load the .env file (optional, especially when running in Docker)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; using environmental variables from system/Docker.")
	}

	dns := os.Getenv("DB_CONNECTION_STR")
	secretKet := os.Getenv("SECRET_KEY")
	s3BucketName := os.Getenv("S3BucketName")
	s3Region := os.Getenv("S3Region")
	s3ApiKey := os.Getenv("S3ApiKey")
	s3Secret := os.Getenv("S3Secret")
	s3Domain := os.Getenv("S3Domain")
	s3upProvider := uploadprovider.NewS3Provider(s3BucketName, s3Region, s3ApiKey, s3Secret, s3Domain)
	db := ConnectToDB(dns)

	// get Sender
	appName := os.Getenv("SENDER_APP_NAME")
	appEmailPw := os.Getenv("SENDER_APP_PW")
	appEmailAdd := os.Getenv("SENDER_EMAIL_ADDRESS")
	gmailSender := mailprovider.NewGmailSender(appName, appEmailAdd, appEmailPw)
	//get oauth
	oauthProvider := oauthprovider.NewGGOAuthProvider(os.Getenv("GOOGLE_CLIENT_ID"),
		os.Getenv("GOOGLE_CLIENT_SECRET"),
		fmt.Sprintf("%s/api/auth/google/callback", os.Getenv("SITE_DOMAIN")),
		[]string{"profile", "email"})

	// get cronjob
	cronProvider := cronjob.NewCronJob()
	cronProvider.StartJobs()

	// get send notification
	fbProjectId := os.Getenv("FIREBASE_PROJECT_ID")
	fbPrivateKeyID := os.Getenv("FIREBASE_PRIVATE_KEY_ID")
	fbPrivateKey := os.Getenv("FIREBASE_PRIVATE_KEY")
	fbClientEmail := os.Getenv("FIREBASE_CLIENT_EMAIL")
	fbClientId := os.Getenv("FIREBASE_CLIENT_ID")

	var sendNotificationProvider sendnotificationprovider.NotificationProvider

	if fbProjectId == "" {
		fmt.Println("Firebase project ID not found; push notifications disabled (No-Op).")
		sendNotificationProvider = sendnotificationprovider.NewNoOpNotificationService()
	} else {
		var err error
		sendNotificationProvider, err = sendnotificationprovider.NewNotificationService(
			context.Background(),
			fbProjectId,
			fbPrivateKeyID,
			fbPrivateKey,
			fbClientEmail,
			fbClientId)

		if err != nil {
			fmt.Println("Failed to initialize Firebase:", err)
			fmt.Println("Fallback to No-Op notification provider.")
			sendNotificationProvider = sendnotificationprovider.NewNoOpNotificationService()
		}
	}

	runServer(db, secretKet, s3upProvider, gmailSender, oauthProvider, cronProvider, sendNotificationProvider)
}
