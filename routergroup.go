// Copyright 2014 Manu Martinez-Almeida. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package gin

import (
	"net/http"
	"path"
	"regexp"
	"strings"
)

var (
	// regEnLetter matches english letters for http method name
	regEnLetter = regexp.MustCompile("^[A-Z]+$")

	// anyMethods for RouterGroup Any method
	anyMethods = []string{
		http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodHead, http.MethodOptions, http.MethodDelete, http.MethodConnect,
		http.MethodTrace,
	}
)

// IRouter defines all router handle interface includes single and group router.
// 链式路由器接口，包含单路由器和分组路由器。
type IRouter interface {
	IRoutes
	Group(string, ...HandlerFunc) *RouterGroup
}

// IRoutes defines all router handle interface.
// 分组嵌套路由器接口，包含单路由器和分组路由器。
type IRoutes interface {
	Use(...HandlerFunc) IRoutes

	Handle(string, string, ...HandlerFunc) IRoutes
	Any(string, ...HandlerFunc) IRoutes
	GET(string, ...HandlerFunc) IRoutes
	POST(string, ...HandlerFunc) IRoutes
	DELETE(string, ...HandlerFunc) IRoutes
	PATCH(string, ...HandlerFunc) IRoutes
	PUT(string, ...HandlerFunc) IRoutes
	OPTIONS(string, ...HandlerFunc) IRoutes
	HEAD(string, ...HandlerFunc) IRoutes
	Match([]string, string, ...HandlerFunc) IRoutes

	StaticFile(string, string) IRoutes
	StaticFileFS(string, string, http.FileSystem) IRoutes
	Static(string, string) IRoutes
	StaticFS(string, http.FileSystem) IRoutes
}

// RouterGroup is used internally to configure router, a RouterGroup is associated with
// a prefix and an array of handlers (middleware).
// RouterGroup 是路由组，包含路由器、中间件和配置设置，（非常基础、重要、核心的配置）
type RouterGroup struct {
	Handlers HandlersChain // 处理函数链，用于存储中间件和处理函数
	basePath string        // 基础路径，用于存储路由组的路径
	engine   *Engine       // 引擎，用于存储引擎实例
	root     bool          // 是否是根路由组
}

var _ IRouter = (*RouterGroup)(nil)

// Use adds middleware to the group, see example code in GitHub.
func (group *RouterGroup) Use(middleware ...HandlerFunc) IRoutes {
	group.Handlers = append(group.Handlers, middleware...)
	return group.returnObj()
}

// Group creates a new router group. You should add all the routes that have common middlewares or the same path prefix.
// For example, all the routes that use a common middleware for authorization could be grouped.
func (group *RouterGroup) Group(relativePath string, handlers ...HandlerFunc) *RouterGroup {
	return &RouterGroup{
		Handlers: group.combineHandlers(handlers),
		basePath: group.calculateAbsolutePath(relativePath),
		engine:   group.engine,
	}
}

// BasePath returns the base path of router group.
// For example, if v := router.Group("/rest/n/v1/api"), v.BasePath() is "/rest/n/v1/api".
func (group *RouterGroup) BasePath() string {
	return group.basePath
}

// 处理路由注册
func (group *RouterGroup) handle(httpMethod, relativePath string, handlers HandlersChain) IRoutes {
	// 将分组的 basePath 和当前路由的 relativePath 拼接成完整路径。
	absolutePath := group.calculateAbsolutePath(relativePath)
	// 将分组的 handlers 和当前路由的 handlers 合并
	handlers = group.combineHandlers(handlers)
	// 把路由注册到路由树中
	group.engine.addRoute(httpMethod, absolutePath, handlers)
	// 返回路由组对象
	return group.returnObj()
}

// Handle registers a new request handle and middleware with the given path and method.
// The last handler should be the real handler, the other ones should be middleware that can and should be shared among different routes.
// See the example code in GitHub.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (group *RouterGroup) Handle(httpMethod, relativePath string, handlers ...HandlerFunc) IRoutes {
	if matched := regEnLetter.MatchString(httpMethod); !matched {
		panic("http method " + httpMethod + " is not valid")
	}
	return group.handle(httpMethod, relativePath, handlers)
}

// POST is a shortcut for router.Handle("POST", path, handlers).
func (group *RouterGroup) POST(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodPost, relativePath, handlers)
}

// GET is a shortcut for router.Handle("GET", path, handlers).
// 本质上是 router.Handle("GET", path, handlers) 的封装
func (group *RouterGroup) GET(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodGet, relativePath, handlers)
}

// DELETE is a shortcut for router.Handle("DELETE", path, handlers).
func (group *RouterGroup) DELETE(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodDelete, relativePath, handlers)
}

// PATCH is a shortcut for router.Handle("PATCH", path, handlers).
func (group *RouterGroup) PATCH(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodPatch, relativePath, handlers)
}

// PUT is a shortcut for router.Handle("PUT", path, handlers).
func (group *RouterGroup) PUT(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodPut, relativePath, handlers)
}

// OPTIONS is a shortcut for router.Handle("OPTIONS", path, handlers).
func (group *RouterGroup) OPTIONS(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodOptions, relativePath, handlers)
}

// HEAD is a shortcut for router.Handle("HEAD", path, handlers).
func (group *RouterGroup) HEAD(relativePath string, handlers ...HandlerFunc) IRoutes {
	return group.handle(http.MethodHead, relativePath, handlers)
}

// Any registers a route that matches all the HTTP methods.
// GET, POST, PUT, PATCH, HEAD, OPTIONS, DELETE, CONNECT, TRACE.
func (group *RouterGroup) Any(relativePath string, handlers ...HandlerFunc) IRoutes {
	for _, method := range anyMethods {
		group.handle(method, relativePath, handlers)
	}

	return group.returnObj()
}

// Match registers a route that matches the specified methods that you declared.
func (group *RouterGroup) Match(methods []string, relativePath string, handlers ...HandlerFunc) IRoutes {
	for _, method := range methods {
		group.handle(method, relativePath, handlers)
	}

	return group.returnObj()
}

// StaticFile registers a single route in order to serve a single file of the local filesystem.
// router.StaticFile("favicon.ico", "./resources/favicon.ico")
func (group *RouterGroup) StaticFile(relativePath, filepath string) IRoutes {
	return group.staticFileHandler(relativePath, func(c *Context) {
		c.File(filepath)
	})
}

// StaticFileFS works just like `StaticFile` but a custom `http.FileSystem` can be used instead..
// router.StaticFileFS("favicon.ico", "./resources/favicon.ico", Dir{".", false})
// Gin by default uses: gin.Dir()
func (group *RouterGroup) StaticFileFS(relativePath, filepath string, fs http.FileSystem) IRoutes {
	return group.staticFileHandler(relativePath, func(c *Context) {
		c.FileFromFS(filepath, fs)
	})
}

func (group *RouterGroup) staticFileHandler(relativePath string, handler HandlerFunc) IRoutes {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static file")
	}
	group.GET(relativePath, handler)
	group.HEAD(relativePath, handler)
	return group.returnObj()
}

// Static serves files from the given file system root.
// Internally a http.FileServer is used, therefore http.NotFound is used instead
// of the Router's NotFound handler.
// To use the operating system's file system implementation,
// use :
//
//	router.Static("/static", "/var/www")
func (group *RouterGroup) Static(relativePath, root string) IRoutes {
	return group.StaticFS(relativePath, Dir(root, false))
}

// StaticFS works just like `Static()` but a custom `http.FileSystem` can be used instead.
// Gin by default uses: gin.Dir()
func (group *RouterGroup) StaticFS(relativePath string, fs http.FileSystem) IRoutes {
	if strings.Contains(relativePath, ":") || strings.Contains(relativePath, "*") {
		panic("URL parameters can not be used when serving a static folder")
	}
	handler := group.createStaticHandler(relativePath, fs)
	urlPattern := path.Join(relativePath, "/*filepath")

	// Register GET and HEAD handlers
	group.GET(urlPattern, handler)
	group.HEAD(urlPattern, handler)
	return group.returnObj()
}

func (group *RouterGroup) createStaticHandler(relativePath string, fs http.FileSystem) HandlerFunc {
	absolutePath := group.calculateAbsolutePath(relativePath)
	fileServer := http.StripPrefix(absolutePath, http.FileServer(fs))

	return func(c *Context) {
		if _, noListing := fs.(*OnlyFilesFS); noListing {
			c.Writer.WriteHeader(http.StatusNotFound)
		}

		file := c.Param("filepath")
		// Check if file exists and/or if we have permission to access it
		f, err := fs.Open(file)
		if err != nil {
			c.Writer.WriteHeader(http.StatusNotFound)
			c.handlers = group.engine.noRoute
			// Reset index
			c.index = -1
			return
		}
		f.Close()

		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}

func (group *RouterGroup) combineHandlers(handlers HandlersChain) HandlersChain {
	finalSize := len(group.Handlers) + len(handlers)
	assert1(finalSize < int(abortIndex), "too many handlers")
	mergedHandlers := make(HandlersChain, finalSize)
	copy(mergedHandlers, group.Handlers)
	copy(mergedHandlers[len(group.Handlers):], handlers)
	return mergedHandlers
}

// 将分组的 basePath 和当前路由的 relativePath 拼接成完整路径。
func (group *RouterGroup) calculateAbsolutePath(relativePath string) string {
	return joinPaths(group.basePath, relativePath)
}

func (group *RouterGroup) returnObj() IRoutes {
	if group.root {
		return group.engine
	}
	return group
}
