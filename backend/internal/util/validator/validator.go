package validator

import (
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

// InitValidator は、カスタムルールを登録した validator を初期化する関数。
func InitValidator() *validator.Validate {
	validate := validator.New()
	RegisterUserRequestValidations(validate)
	return validate
}

// isAlphaNumStartingWithLetter は、英数字のみを許可し、
// 先頭は英字であることをチェックする関数。
func isAlphaNumStartingWithLetter(field validator.FieldLevel) bool {
	return matchRegexp("^[A-Za-z][A-Za-z0-9]+$", field.Field().String())
}

// isAlphaNumericWithMixedCaseAndNumber は、英数字のみを許可し、
// 小文字・大文字・数字をそれぞれ1文字以上含むかをチェックする関数。
func isAlphaNumericWithMixedCaseAndNumber(field validator.FieldLevel) bool {
	password := field.Field().String()

	hasLower := regexp.MustCompile("[a-z]").MatchString(password)
	hasUpper := regexp.MustCompile("[A-Z]").MatchString(password)
	hasNumber := regexp.MustCompile("[0-9]").MatchString(password)
	isAlphaNum := regexp.MustCompile("^[A-Za-z0-9]+$").MatchString(password)

	return hasLower && hasUpper && hasNumber && isAlphaNum
}

// matchRegexp は、正規表現 pattern に文字列 value が一致するかを判定する関数。
func matchRegexp(pattern, value string) bool {
	return regexp.MustCompile(pattern).Match([]byte(value))
}

// RegisterUserRequestValidations は、パスワード用のカスタムルールを登録する関数。
func RegisterUserRequestValidations(validate *validator.Validate) {
	validate.RegisterValidation("username_format", isAlphaNumStartingWithLetter)
	validate.RegisterValidation("password_format", isAlphaNumericWithMixedCaseAndNumber)
}

// TranslateError は、validator のエラーをユーザー向けのメッセージに変換する関数。
func TranslateError(err error) string {
	if errs, ok := err.(validator.ValidationErrors); ok {
		for _, err := range errs {
			switch err.Tag() {
			case "required":
				return fmt.Sprintf("%s は必須項目です。", err.Field())
			case "min":
				return fmt.Sprintf("%s は %s 文字以上にしてください。", err.Field(), err.Param())
			case "max":
				return fmt.Sprintf("%s は %s 文字以下にしてください。", err.Field(), err.Param())
			case "username_format":
				return "ユーザー名は英字で始まり、英数字のみ使用してください。"
			case "password_format":
				return "パスワードは小文字・大文字・数字をそれぞれ1文字以上を含め、英数字のみ使用してください。"
			default:
				return fmt.Sprintf("%s の入力が正しくありません。", err.Field())
			}
		}
	}
	return "入力内容に誤りがあります。"
}