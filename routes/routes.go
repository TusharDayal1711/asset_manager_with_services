package routes

import (
	"asset/handler/assetHandler"
	"asset/handler/userHandler"
	"asset/middlewareprovider"
	"asset/models"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type RouteHandler struct {
	UserHandler    *userhandler.UserHandler
	AssetHandler   *assethandler.AssetHandler
	AuthMiddleware middlewareprovider.AuthMiddlewareService
}

func RegisterRoutes(h RouteHandler) http.Handler {
	r := chi.NewRouter()

	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("connection established..."))
	})

	r.Route("/api", func(r chi.Router) {
		//public
		r.Post("/user/register", h.UserHandler.PublicRegister)
		r.Post("/user/login", h.UserHandler.UserLogin)

		//protected
		r.Group(func(protected chi.Router) {
			protected.Use(h.AuthMiddleware.JWTAuthMiddleware())

			protected.Get("/users/dashboard", h.UserHandler.GetUserDashboard)

			//asset manager and admin
			protected.Route("/inventory", func(ir chi.Router) {
				ir.Use(h.AuthMiddleware.RequireRole(models.AssetManagerRole, models.AdminRole)) // ✅ Updated

				// POST
				ir.Post("/asset", h.AssetHandler.AddNewAssetWithConfig)
				ir.Post("/asset/assign", h.AssetHandler.AssignAssetToUser)
				ir.Post("/asset/unassign", h.AssetHandler.RetrieveAsset)
				ir.Post("/asset/service/send", h.AssetHandler.SendAssetToService)
				ir.Post("/asset/service/received", h.AssetHandler.ReceivedFromService)

				//put
				ir.Put("/asset/update", h.AssetHandler.UpdateAssetWithConfigHandler)

				//get
				ir.Get("/assets", h.AssetHandler.GetAllAssetsWithFilters)
				ir.Get("/asset/timeline", h.AssetHandler.GetAssetTimeline)

				//delete
				ir.Delete("/asset/remove", h.AssetHandler.DeleteAsset)
			})

			//employee manager and admin
			protected.Route("/employee", func(er chi.Router) {
				er.Use(h.AuthMiddleware.RequireRole(models.EmployeeMangerRole, models.AdminRole)) // ✅ Updated

				//post
				er.Post("/register", h.UserHandler.RegisterEmployeeByManager)

				//put
				er.Put("/update", h.UserHandler.UpdateEmployee)

				//get
				er.Get("/employees", h.UserHandler.GetEmployeesWithFilters)
				er.Get("/timeline", h.UserHandler.GetEmployeeTimeline)

				//delete
				er.Delete("/remove", h.UserHandler.DeleteUser)
			})

			//admin only
			protected.Route("/admin", func(ar chi.Router) {
				ar.Use(h.AuthMiddleware.RequireRole(models.AdminRole))
				ar.Post("/employee/change-permissions", h.UserHandler.ChangeUserRole)
			})
		})
	})

	return r
}
