package validation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strings"

	appErrors "educore/internal/shared/errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func init() {
	// Register tag name function to extract json tag name instead of Go struct field name
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			return fld.Name
		}
		return name
	})
}

// BindJSON parses and strictly validates a JSON request body from gin.Context into obj.
// It enforces:
// 1. Content-Type must be application/json
// 2. Non-empty body
// 3. Disallow unknown fields
// 4. Exactly one JSON document (no trailing payload)
// 5. Struct validation rules via validator/v10
func BindJSON(c *gin.Context, obj any) error {
	// Check Content-Type
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		return appErrors.ErrUnsupportedMediaType("Header Content-Type wajib disertakan dan harus application/json")
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "application/json" {
		return appErrors.ErrUnsupportedMediaType("Content-Type request tidak didukung, gunakan application/json")
	}

	// Read body to memory to verify non-empty and allow inspection
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return appErrors.ErrContentTooLarge("")
		}
		return appErrors.ErrBadRequest("Gagal membaca body request")
	}

	// Restore body for any subsequent readers if needed
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	trimmedBody := bytes.TrimSpace(bodyBytes)
	if len(trimmedBody) == 0 {
		return appErrors.ErrBadRequest("Body request tidak boleh kosong")
	}

	// Strict JSON decoding with DisallowUnknownFields
	dec := json.NewDecoder(bytes.NewReader(trimmedBody))
	dec.DisallowUnknownFields()

	if err := dec.Decode(obj); err != nil {
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxErr):
			return appErrors.ErrBadRequest(fmt.Sprintf("Format JSON tidak valid pada karakter %d", syntaxErr.Offset))
		case errors.As(err, &unmarshalTypeErr):
			field := unmarshalTypeErr.Field
			if field == "" {
				field = "payload"
			}
			msg := fmt.Sprintf("Tipe data untuk field '%s' tidak valid (diharapkan %s, diterima %s)", field, unmarshalTypeErr.Type.String(), unmarshalTypeErr.Value)
			return appErrors.ErrValidation([]appErrors.Detail{
				{Field: field, Message: msg},
			})
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			fieldName = strings.Trim(fieldName, `"`)
			return appErrors.ErrBadRequest(fmt.Sprintf("Field '%s' tidak dikenal dan tidak diperbolehkan", fieldName))
		default:
			return appErrors.ErrBadRequest(fmt.Sprintf("Body JSON rusak atau tidak sesuai: %v", err))
		}
	}

	// Ensure there is only one JSON document
	var extra json.RawMessage
	if err := dec.Decode(&extra); err != io.EOF {
		return appErrors.ErrBadRequest("Body request hanya boleh berisi satu dokumen JSON")
	}

	// Validate struct fields
	return ValidateStruct(obj)
}

// ValidateStruct validates a struct instance and converts validator.ValidationErrors into *appErrors.AppError.
func ValidateStruct(obj any) error {
	if obj == nil {
		return nil
	}

	err := validate.Struct(obj)
	if err == nil {
		return nil
	}

	var valErrs validator.ValidationErrors
	if errors.As(err, &valErrs) {
		details := make([]appErrors.Detail, 0, len(valErrs))
		for _, fe := range valErrs {
			details = append(details, appErrors.Detail{
				Field:   fe.Field(),
				Message: formatFieldError(fe),
			})
		}
		return appErrors.ErrValidation(details)
	}

	return appErrors.ErrBadRequest(fmt.Sprintf("Validasi gagal: %v", err))
}

func formatFieldError(fe validator.FieldError) string {
	field := fe.Field()
	param := fe.Param()

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("Field '%s' wajib diisi", field)
	case "email":
		return fmt.Sprintf("Format '%s' harus berupa alamat email yang valid", field)
	case "min":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("Panjang '%s' minimal %s karakter", field, param)
		}
		return fmt.Sprintf("Nilai '%s' minimal %s", field, param)
	case "max":
		if fe.Kind() == reflect.String {
			return fmt.Sprintf("Panjang '%s' maksimal %s karakter", field, param)
		}
		return fmt.Sprintf("Nilai '%s' maksimal %s", field, param)
	case "gte":
		return fmt.Sprintf("Nilai '%s' harus lebih besar atau sama dengan %s", field, param)
	case "lte":
		return fmt.Sprintf("Nilai '%s' harus lebih kecil atau sama dengan %s", field, param)
	case "gt":
		return fmt.Sprintf("Nilai '%s' harus lebih besar dari %s", field, param)
	case "lt":
		return fmt.Sprintf("Nilai '%s' harus lebih kecil dari %s", field, param)
	case "oneof":
		return fmt.Sprintf("Nilai '%s' harus salah satu dari: [%s]", field, param)
	case "alphanum":
		return fmt.Sprintf("Field '%s' hanya boleh berisi karakter alfanumerik", field)
	case "numeric":
		return fmt.Sprintf("Field '%s' hanya boleh berisi angka", field)
	case "uuid":
		return fmt.Sprintf("Format '%s' harus berupa UUID yang valid", field)
	case "datetime":
		return fmt.Sprintf("Format tanggal/waktu '%s' tidak valid (harus sesuai %s)", field, param)
	default:
		return fmt.Sprintf("Field '%s' tidak memenuhi aturan validasi '%s'", field, fe.Tag())
	}
}
