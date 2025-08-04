package server

import (
	"asset/models"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
)

func (srv *Server) InjectRoutes() *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("connection established..."))
	})

	//public routes
	r.Route("/api", func(api chi.Router) {
		api.Post("/user/register", srv.UserHandler.PublicRegister)
		api.Post("/v2/user/register", srv.UserHandler.PublicRegisterThroughFirebase)
		api.Post("/user/login", srv.UserHandler.UserLogin)
		api.Post("/v2/user/login", srv.UserHandler.GoogleAuth)
		//api.Post("/createadmin", srv.UserHandler.CreateAdmin)

		//protected
		api.Group(func(protected chi.Router) {
			protected.Use(srv.Middleware.JWTAuthMiddleware())

			protected.Get("/users/dashboard", srv.UserHandler.GetUserDashboard)

			//asset_manage and admin routes
			protected.Route("/inventory", func(inventory chi.Router) {
				inventory.Use(srv.Middleware.RequireRole(models.AssetManagerRole, models.AdminRole))

				//post methods
				inventory.Post("/asset", srv.AssetHandler.AddNewAssetWithConfig)
				inventory.Post("/asset/assign", srv.AssetHandler.AssignAssetToUser)
				inventory.Post("/asset/unassign", srv.AssetHandler.RetrieveAsset)
				inventory.Post("/asset/service/send", srv.AssetHandler.SendAssetToService)
				inventory.Post("/asset/service/received", srv.AssetHandler.ReceivedFromService)

				//put methods
				inventory.Put("/asset/update", srv.AssetHandler.UpdateAssetWithConfigHandler)

				//get methods
				inventory.Get("/assets", srv.AssetHandler.GetAllAssetsWithFilters)
				inventory.Get("/asset/timeline", srv.AssetHandler.GetAssetTimeline)

				//delete methods
				inventory.Delete("/asset/remove", srv.AssetHandler.DeleteAsset)
			})

			//employee_manager and admin routes
			protected.Route("/employee", func(employee chi.Router) {
				employee.Use(srv.Middleware.RequireRole(models.EmployeeMangerRole, models.AdminRole))

				//post methods
				employee.Post("/register", srv.UserHandler.RegisterEmployeeByManager)

				//put methods
				employee.Put("/update", srv.UserHandler.UpdateEmployee)

				//get methods
				employee.Get("/employees", srv.UserHandler.GetEmployeesWithFilters)
				employee.Get("/timeline", srv.UserHandler.GetEmployeeTimeline)

				//delete methods
				employee.Delete("/remove", srv.UserHandler.DeleteUser)
			})

			// Admin-only routes
			protected.Route("/admin", func(admin chi.Router) {
				admin.Use(srv.Middleware.RequireRole(models.AdminRole))
				admin.Post("/employee/change-permissions", srv.UserHandler.ChangeUserRole)
			})
		})
	})

	return r
}
