package handler

import (
	"fmt"
	"net/http"
)

func HTTPInterceptor(h http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			username := r.Form.Get("username")
			token := r.Form.Get("token")

			if len(username) < 3 || !IsTokenValid(token, username) {
				fmt.Println("拦截器Inceptor工作")
				w.WriteHeader(http.StatusForbidden)
				return
			}
			h(w, r)
		},
	)
}
