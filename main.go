package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/dchest/captcha"
	"golang.org/x/exp/rand"

	"github.com/gofiber/fiber/v2"

	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"github.com/uptrace/bun/extra/bundebug"
)

var (
	DB         *bun.DB
	Ctx        = context.Background()
	captchaDir = "captchas"
)

type Captcha struct {
	bun.BaseModel `bun:"table:player,alias:p"`

	ID        uint64    `bun:",pk,autoincrement"`
	IP        string    `bun:",notnull"`
	Token     string    `bun:",notnull"`
	Value     string    `bun:",notnull"`
	Timestamp time.Time `bun:",notnull,default:current_timestamp"`
}

type Contact struct {
	Email   string `json:"email"`
	Captcha string `json:"captcha"`
}

func RandomString(length int) (string, error) {
	randomBytes := make([]byte, length)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	randomString := base64.URLEncoding.EncodeToString(randomBytes)

	return randomString, nil
}

func deleteOldCaptcha(minutes int) error {
	threshold := time.Now().Add(-time.Duration(minutes) * time.Minute)

	return filepath.Walk(captchaDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && info.ModTime().Before(threshold) {
			fmt.Println("Deleting:", path)
			return os.Remove(path)
		}

		return nil
	})
}

func main() {
	if err := os.MkdirAll("captchas", os.ModePerm); err != nil {
		panic(err)
	}

	sqldb, err := sql.Open(sqliteshim.ShimName, "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	DB := bun.NewDB(sqldb, sqlitedialect.New())

	file, err := os.Create("bundebug.log")
	if err != nil {
		panic(err)
	}

	DB.AddQueryHook(bundebug.NewQueryHook(
		bundebug.WithVerbose(true),
		bundebug.WithWriter(file),
	))

	if err := DB.Ping(); err != nil {
		panic(err)
	}

	DB.RegisterModel(&Captcha{})
	if _, err = DB.NewCreateTable().Model(&Captcha{}).Exec(Ctx); err != nil {
		panic(err)
	}

	app := fiber.New()

	app.Post("/contact", func(c *fiber.Ctx) error {
		captchaToken := c.Cookies("captcha_token")
		ipHash := sha256.Sum256([]byte(c.IP()))
		ip := fmt.Sprintf("%x", ipHash)

		captchaEntity := Captcha{}
		getCaptcha := DB.NewSelect().Model(&captchaEntity)
		getCaptcha.Where("token = ? AND ip = ? AND timestamp > ?", captchaToken, ip, time.Now().Add(-2*time.Minute))
		if err := getCaptcha.Scan(Ctx); err != nil {
			return c.SendStatus(404)
		}

		var contactForm Contact
		if err := c.BodyParser(&contactForm); err != nil {
			return c.SendStatus(400)
		}

		if !captcha.VerifyString(captchaEntity.Value, contactForm.Captcha) {
			return c.SendStatus(403)
		}

		return c.SendStatus(200)
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("contact.html")
	})

	app.Get("/captcha", func(c *fiber.Ctx) error {
		token, err := RandomString(32)
		if err != nil {
			return c.SendStatus(400)
		}

		_ = deleteOldCaptcha(2)

		captchaCookie := new(fiber.Cookie)
		captchaCookie.Name = "captcha_token"
		captchaCookie.Value = token
		captchaCookie.Expires = time.Now().Add(2 * time.Minute)
		captchaCookie.HTTPOnly = true
		captchaCookie.Secure = true
		c.Cookie(captchaCookie)

		captchaId := captcha.New()

		ipHash := sha256.Sum256([]byte(c.IP()))
		ip := fmt.Sprintf("%x", ipHash)

		captchaEntity := &Captcha{
			IP:    ip,
			Token: token,
			Value: captchaId,
		}

		insertCaptcha := DB.NewInsert().Model(captchaEntity)
		if _, err = insertCaptcha.Exec(Ctx); err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to create captcha entity")
		}

		captchaPath := filepath.Join("captchas", captchaId+".png")

		// create file writer
		file, err := os.Create(captchaPath)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to create captcha file")
		}
		defer file.Close()

		// Write the captcha image
		if err = captcha.WriteImage(file, captchaId, 240, 80); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to write captcha image")
		}

		return c.SendFile(captchaPath, true)
	})

	log.Fatal(app.Listen(":3000"))
}
