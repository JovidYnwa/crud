package middleware

import(
	"net/http"
	"github.com/JovidYnwa/crud/pkg/security"
)

func Basic(securitySvc *security.Service) func(handler http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request){
			login, password, _:= request.BasicAuth()

			isAuth := securitySvc.Auth(request.Context(), login, password)

			if !isAuth {
				http.Error(writer, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			
			handler.ServeHTTP(writer, request)
		})
	}
}