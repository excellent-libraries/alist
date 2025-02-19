package server

import (
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/message"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/alist-org/alist/v3/server/controllers"
	"github.com/alist-org/alist/v3/server/middlewares"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Init(r *gin.Engine) {
	common.SecretKey = []byte(conf.Conf.JwtSecret)
	Cors(r)
	WebDav(r)

	r.GET("/d/*path", middlewares.Down, controllers.Down)
	r.GET("/p/*path", middlewares.Down, controllers.Proxy)

	r.POST("/api/auth/login", controllers.Login)

	api := r.Group("/api", middlewares.Auth)
	api.GET("/auth/current", controllers.CurrentUser)

	admin := api.Group("/admin", middlewares.AuthAdmin)

	meta := admin.Group("/meta")
	meta.GET("/list", controllers.ListMetas)
	meta.POST("/create", controllers.CreateMeta)
	meta.POST("/update", controllers.UpdateMeta)
	meta.POST("/delete", controllers.DeleteMeta)

	user := admin.Group("/user")
	user.GET("/list", controllers.ListUsers)
	user.POST("/create", controllers.CreateUser)
	user.POST("/update", controllers.UpdateUser)
	user.POST("/delete", controllers.DeleteUser)

	account := admin.Group("/account")
	account.GET("/list", controllers.ListAccounts)
	account.POST("/create", controllers.CreateAccount)
	account.POST("/update", controllers.UpdateAccount)
	account.POST("/delete", controllers.DeleteAccount)

	driver := admin.Group("/driver")
	driver.GET("/list", controllers.ListDriverItems)
	driver.GET("/names", controllers.ListDriverNames)
	driver.GET("/items", controllers.GetDriverItems)

	setting := admin.Group("/setting")
	setting.GET("/get", controllers.GetSetting)
	setting.GET("/list", controllers.ListSettings)
	setting.POST("/save", controllers.SaveSettings)
	setting.POST("/delete", controllers.DeleteSetting)
	setting.POST("/reset_token", controllers.ResetToken)
	setting.POST("/set_aria2", controllers.SetAria2)

	task := admin.Group("/task")
	task.GET("/down/undone", controllers.UndoneDownTask)
	task.GET("/down/done", controllers.DoneDownTask)
	task.POST("/down/cancel", controllers.CancelDownTask)
	task.GET("/transfer/undone", controllers.UndoneTransferTask)
	task.GET("/transfer/done", controllers.DoneTransferTask)
	task.POST("/transfer/cancel", controllers.CancelTransferTask)
	task.GET("/upload/undone", controllers.UndoneUploadTask)
	task.GET("/upload/done", controllers.DoneUploadTask)
	task.POST("/upload/cancel", controllers.CancelUploadTask)
	task.GET("/copy/undone", controllers.UndoneCopyTask)
	task.GET("/copy/done", controllers.DoneCopyTask)
	task.POST("/copy/cancel", controllers.CancelCopyTask)

	ms := admin.Group("/message")
	ms.GET("/get", message.PostInstance.GetHandle)
	ms.POST("/send", message.PostInstance.SendHandle)

	// guest can
	public := api.Group("/public")
	public.GET("/settings", controllers.PublicSettings)
	public.Any("/list", controllers.FsList)
	public.Any("/get", controllers.FsGet)

	// gust can't
	fs := api.Group("/fs")
	fs.POST("/mkdir", controllers.FsMkdir)
	fs.POST("/rename", controllers.FsRename)
	fs.POST("/move", controllers.FsMove)
	fs.POST("/copy", controllers.FsCopy)
	fs.POST("/remove", controllers.FsRemove)
	fs.POST("/put", controllers.FsPut)
	fs.POST("/link", middlewares.AuthAdmin, controllers.Link)
	fs.POST("/add_aria2", controllers.AddAria2)
}

func Cors(r *gin.Engine) {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = append(config.AllowHeaders, "Authorization", "range", "File-Path", "As-Task")
	r.Use(cors.New(config))
}
