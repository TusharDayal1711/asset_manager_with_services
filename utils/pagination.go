package utils

import (
	"net/http"
	"strconv"
)

func GetPageLimitAndOffset(r *http.Request) (int, int) {
	page := 1
	limit := 10
	if pageValue := r.URL.Query().Get("page"); pageValue != "" {
		if p, err := strconv.Atoi(pageValue); err == nil {
			page = p //if error ocurs keep the default valuse
		}
	}
	if limitValue := r.URL.Query().Get("limit"); limitValue != "" {
		if l, err := strconv.Atoi(limitValue); err == nil {
			limit = l //if error ocurs keep the default valuse
		}
	}
	offset := (page - 1) * limit
	return limit, offset
}
