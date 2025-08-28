package main

func init() {
	initLogger()
	printBanner()
	initHTTPClient()
	initACME()
	initRoutes()
}

func main() {
	go serveManager()
	Log.Println("[HTTP] Listening and serving HTTP on " + addr + ":" + port)
	r.Run(addr + ":" + port)
}
