package form

type Contact struct {
	Email       string `json:"email"`
	Captcha     string `json:"captcha"`
	Description string `json:"description"`
}
