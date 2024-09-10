package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"kevyn.fr/contact/app/db"
	"kevyn.fr/contact/app/form"
	"kevyn.fr/contact/app/mail"
	"kevyn.fr/contact/app/service/captcha"
	"kevyn.fr/contact/app/utils"

	"github.com/gofiber/fiber/v2"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db.Init()
	mail.Init()

	if err := os.MkdirAll("captchas", os.ModePerm); err != nil {
		panic(err)
	}

	app := fiber.New()

	app.Post("/contact", func(c *fiber.Ctx) error {
		var contactForm form.Contact
		if err := c.BodyParser(&contactForm); err != nil {
			return c.SendStatus(400)
		}

		captchaToken := c.Cookies("captcha_token")
		ipHash := sha256.Sum256([]byte(c.IP()))
		ip := fmt.Sprintf("%x", ipHash)

		if !captcha.Verify(captchaToken, ip, contactForm) {
			return c.SendStatus(403)
		}

		// send contact form as mail
		if err := mail.SendMail(fmt.Sprintf("Formulaire de contact de %s", contactForm.Email), contactForm.Description); err != nil {
			return c.SendStatus(400)
		}

		return c.SendStatus(200)
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("contact.html")
	})

	app.Get("/captcha", func(c *fiber.Ctx) error {
		token, err := utils.RandomString(32)
		if err != nil {
			return c.SendStatus(400)
		}

		_ = captcha.DeleteOldFiles(2)

		captchaCookie := new(fiber.Cookie)
		captchaCookie.Name = "captcha_token"
		captchaCookie.Value = token
		captchaCookie.Expires = time.Now().Add(2 * time.Minute)
		captchaCookie.HTTPOnly = true
		captchaCookie.Secure = true
		c.Cookie(captchaCookie)

		ipHash := sha256.Sum256([]byte(c.IP()))
		ip := fmt.Sprintf("%x", ipHash)

		captchaPath, err := captcha.Create(ip, token)

		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Failed to create captcha entity")
		}

		return c.SendFile(captchaPath, false)
	})

	log.Fatal(app.Listen(":3000"))
}
