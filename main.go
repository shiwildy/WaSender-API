package main

import (
	"context"
	"os/exec"
	"time"
	"net/http"
	"runtime"
	"go.wasender.api/helper"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log"
	"path/filepath"
	"os"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	// "go.mau.fi/whatsmeow/types"
)

func init() {
	store.DeviceProps.PlatformType = waProto.DeviceProps_ANDROID_PHONE.Enum() // set props to android device
	store.DeviceProps.Os = proto.String("WaSender-API")
}

func clearScreen() {
	if runtime.GOOS == "windows" {
		c := exec.Command("cls")
		c.Stdout = os.Stdout
		c.Run()	
	} else {
		c := exec.Command("clear")
		c.Stdout = os.Stdout
		c.Run()
	}
}

func AutoDeleteFiles() {
	log.Println("Starting Temporary cleaner goroutine")
	interval := 5 * time.Minute
	timer := time.NewTimer(interval)

	for {
		select {
		case <-timer.C:
			err := deleteTempFiles()
			if err != nil {
				log.Println("Failed to delete temp files:", err)
			}

			timer.Reset(interval)
		}
	}
}

func deleteTempFiles() error {
	dir, err := filepath.Abs("temp/")
	if err != nil {
		return err
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		err := os.Remove(filepath.Join(dir, file.Name()))
		if err != nil {
			log.Println("Failed to delete file:", err)
		}
	}

	return nil
}

func main() {
	// Set gin to release
	gin.SetMode(gin.ReleaseMode)

	// register new gin container
	rgin := gin.New()

	// register sqlite container
	dbLog := waLog.Stdout("Database", "ERROR", true)
	container, err := sqlstore.New("sqlite3", "file:secret/session.db", dbLog)
	if err != nil {
		log.Println("Failed to create sqlite3 container:", err)
		return
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		log.Println("Failed to retrieve stored data from container", err)
		return
	}

	clientLog := waLog.Stdout("Client", "ERROR", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	if client.Store.ID == nil {
		qr_code, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			log.Println("Failed to generate qrcode", err)
			return
		}

		for evt := range qr_code {
			log.Println(evt.Event)
			if evt.Event == "code" {
				clearScreen()
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				log.Println("Scan QRCode to Login !")
			} else if evt.Event == "err-client-outdated" {
				clearScreen()
				log.Println("Client outdated, please upgrade the whatsmeow modules")
				return // return because error in outdated modules
			} else {
				clearScreen()
				log.Println("Connected to whatsapp")

				// Starting goroutine for autodelete temporary folder
				go AutoDeleteFiles()
			}
		}

	} else {
		err = client.Connect()
		if err != nil {
			log.Println("Cannot open connection to whatsapp", err)
			return
		}

		// start go routines for auto delete temp files
		clearScreen()
		log.Println("Connected to whatsapp")

		// Starting goroutine for autodelete temporary folder
		go AutoDeleteFiles()
	}

	// register helper as hexec
	hexec := helper.Register(client)

	// Gin default route and authentication
	rgin.Use(func(c *gin.Context) {
		token := c.GetHeader("Authorization")	
		if token != "Bearer your_secret_token" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
	})

	// handle for /sendtext
	rgin.POST("/sendtext", func(c *gin.Context) {
		var req struct {
			To   string `json:"to"`
			Text string `json:"text"`
		}

		err := c.BindJSON(&req)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		clientIP := c.ClientIP()
		log.Println("Requests received from", clientIP)

		err = hexec.SendMessage(req.To, req.Text)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send message"})
			log.Println("Failed to process request from", clientIP, "to sending a message to", req.To)
			return
		}

		log.Println("Success process request from", clientIP, "to sending a message to", req.To)
		c.JSON(http.StatusOK, gin.H{"message": "message sent successfully"})
	})

	// handle for /senddoc
	rgin.POST("/senddoc", func(c *gin.Context) {
		var req struct {
			To        string `json:"to"`
			Caption   string `json:"caption"`
			Filename  string `json:"filename"`
			Document  []byte `json:"document"`
		}
	
		err := c.BindJSON(&req)
	
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
	
		// Create temporary files
		uuid := uuid.New().String()
		tempFile, err := os.CreateTemp("temp/", uuid)
	
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temporary file"})
			return
		}
	
		defer tempFile.Close()
	
		// Write the document data to the temporary file
		_, err = tempFile.Write(req.Document)
	
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write document to file"})
			return
		}

		clientIP := c.ClientIP()
		log.Println("Requests received from", clientIP)

		// Send the document
		filedir := filepath.Join(tempFile.Name())
		err = hexec.SendDocument(req.To, filedir, req.Caption, req.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send document"})
			log.Println("Failed to process request from", clientIP, "to sending a document to", req.To)
			return
		}
	
		log.Println("Success process request from", clientIP, "to sending a document to", req.To)
		c.JSON(http.StatusOK, gin.H{"message": "document sent successfully"})
	})

	// handle for /sendimg
	rgin.POST("/sendimg", func(c *gin.Context) {
		var req struct {
			To      string `json:"to"`
			Caption string `json:"caption"`
			Image   []byte `json:"image"`
		}

		err := c.BindJSON(&req)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}

		// Create temporary files
		uuid := uuid.New().String()
		tempFile, err := os.CreateTemp("temp/", uuid)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temporary file"})
			return
		}
	
		defer tempFile.Close()

		// Write the document data to the temporary file
		_, err = tempFile.Write(req.Image)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write document to file"})
			return
		}

		clientIP := c.ClientIP()
		log.Println("Requests received from", clientIP)

		// Send the image
		filedir := filepath.Join(tempFile.Name())
		err = hexec.SendImage(req.To, filedir, req.Caption)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send image"})
			log.Println("Failed to process request from", clientIP, "to sending a images to", req.To)
			return
		}

		log.Println("Success process request from", clientIP, "to sending a image to", req.To)
		c.JSON(http.StatusOK, gin.H{"message": "image sent successfully"})
	})

	// listen at 8080
	rgin.Run(":8080")

	// set the c make chain to 1 for fixing should bufferd
	c := make(chan os.Signal, 1)
	<-c

	// disconnecting whatsmeow client
	client.Disconnect()
}
