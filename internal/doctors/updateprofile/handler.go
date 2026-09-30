package updateprofile

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"

	"tibi/internal/platform/auth"
	"tibi/internal/platform/httpx"
)

var v = validator.New()

func init() {
	if err := v.RegisterValidation("decimal_positive", func(fl validator.FieldLevel) bool {
		fee, err := strconv.ParseFloat(fl.Field().String(), 64)
		return err == nil && fee > 0
	}); err != nil {
		panic(err)
	}
	if err := v.RegisterValidation("uppercase", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		return s != "" && s == strings.ToUpper(s)
	}); err != nil {
		panic(err)
	}
}

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		httpx.Error(w, r, httpx.Unauthenticated("not authenticated"))
		return
	}

	var cmd Command
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cmd); err != nil {
		httpx.Error(w, r, httpx.BadRequest("malformed json"))
		return
	}
	if err := v.Struct(cmd); err != nil {
		httpx.Error(w, r, httpx.ValidationFailed(validationDetails(err)))
		return
	}

	resp, err := h.svc.Execute(r.Context(), u.UserID, cmd)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func validationDetails(err error) map[string]string {
	out := map[string]string{}
	if ves, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ves {
			out[fe.Field()] = fe.Tag()
		}
	}
	return out
}
