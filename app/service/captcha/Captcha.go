package captcha

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dchest/captcha"
	"kevyn.fr/contact/app/db"
	"kevyn.fr/contact/app/form"
)

func Create(ip string, token string) (string, error) {
	captchaId := captcha.New()
	captchaEntity := &db.Captcha{
		IP:    ip,
		Token: token,
		Value: captchaId,
	}

	insertCaptcha := db.DB.NewInsert().Model(captchaEntity)
	if _, err := insertCaptcha.Exec(db.Ctx); err != nil {
		return "", err
	}

	captchaPath := filepath.Join("captchas", captchaId+".png")

	// create file writer
	file, err := os.Create(captchaPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Write the captcha image
	if err := captcha.WriteImage(file, captchaId, 240, 80); err != nil {
		return "", err
	}

	return captchaPath, nil
}

func Verify(captchaToken string, ip string, contactForm form.Contact) bool {
	captchaEntity := db.Captcha{}
	getCaptcha := db.DB.NewSelect().Model(&captchaEntity)
	getCaptcha.Where("token = ? AND ip = ? AND timestamp > ?", captchaToken, ip, time.Now().Add(-2*time.Minute))
	if err := getCaptcha.Scan(db.Ctx); err != nil {
		return false
	}

	return captcha.VerifyString(captchaEntity.Value, contactForm.Captcha)
}

func DeleteOldFiles(minutes int) error {
	threshold := time.Now().Add(-time.Duration(minutes) * time.Minute)

	return filepath.Walk("captchas", func(path string, info os.FileInfo, err error) error {
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
