package main

import (
	"github.com/labstack/echo"
	"net/http"
)

type ScrapeRequest struct {
	InfoHashes []string
}

type ScrapeResponse struct {
}

// Route handler for the /scrape requests
func HandleScrape(c *echo.Context) {
	Debug("Got scrape")
	Debug(c.Request.RequestURI)
	c.String(http.StatusOK, "I like to scrape my ass")
}
