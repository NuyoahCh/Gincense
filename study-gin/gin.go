package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// 创建一个默认的路由引擎
	r := gin.Default()
	// 定义路由和处理函数
	r.GET("/ping", func(ctx *gin.Context) {
		// 返回JSON数据
		ctx.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	// 启动HTTP服务器并监听端口
	r.Run()
}