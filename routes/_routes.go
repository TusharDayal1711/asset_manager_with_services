package routes

import (
	"asset/handler/assetHandler"
	"asset/handler/userHandler"
	"asset/models"
	"asset/providers"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type RouteHandler struct {
	UserHandler    *userhandler.UserHandler
	AssetHandler   *assethandler.AssetHandler
	AuthMiddleware providers.AuthMiddlewareService
}

func RegisterRoutes(h RouteHandler) http.Handler {
	mainRouter := chi.NewRouter()

	mainRouter.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("connection established..."))
	})

	mainRouter.Route("/api", func(apiRouter chi.Router) {
		//public routes
		apiRouter.Post("/user/register", h.UserHandler.PublicRegister)
		apiRouter.Post("/user/login", h.UserHandler.UserLogin)

		//protected routes
		apiRouter.Group(func(protectedRouter chi.Router) {
			protectedRouter.Use(h.AuthMiddleware.JWTAuthMiddleware())

			protectedRouter.Get("/users/dashboard", h.UserHandler.GetUserDashboard)

			//routes for asset managers and admins
			protectedRouter.Route("/inventory", func(inventoryRouter chi.Router) {
				inventoryRouter.Use(h.AuthMiddleware.RequireRole(models.AssetManagerRole, models.AdminRole))

				//post
				inventoryRouter.Post("/asset", h.AssetHandler.AddNewAssetWithConfig)
				inventoryRouter.Post("/asset/assign", h.AssetHandler.AssignAssetToUser)
				inventoryRouter.Post("/asset/unassign", h.AssetHandler.RetrieveAsset)
				inventoryRouter.Post("/asset/service/send", h.AssetHandler.SendAssetToService)
				inventoryRouter.Post("/asset/service/received", h.AssetHandler.ReceivedFromService)

				//put
				inventoryRouter.Put("/asset/update", h.AssetHandler.UpdateAssetWithConfigHandler)

				//get
				inventoryRouter.Get("/assets", h.AssetHandler.GetAllAssetsWithFilters)
				inventoryRouter.Get("/asset/timeline", h.AssetHandler.GetAssetTimeline)

				//delete
				inventoryRouter.Delete("/asset/remove", h.AssetHandler.DeleteAsset)
			})

			//routes for employee managers and admins
			protectedRouter.Route("/employee", func(employeeRouter chi.Router) {
				employeeRouter.Use(h.AuthMiddleware.RequireRole(models.EmployeeMangerRole, models.AdminRole))

				//post
				employeeRouter.Post("/register", h.UserHandler.RegisterEmployeeByManager)

				//put
				employeeRouter.Put("/update", h.UserHandler.UpdateEmployee)

				//get
				employeeRouter.Get("/employees", h.UserHandler.GetEmployeesWithFilters)
				employeeRouter.Get("/timeline", h.UserHandler.GetEmployeeTimeline)

				//delete
				employeeRouter.Delete("/remove", h.UserHandler.DeleteUser)
			})

			//routes for admins only
			protectedRouter.Route("/admin", func(adminRouter chi.Router) {
				adminRouter.Use(h.AuthMiddleware.RequireRole(models.AdminRole))
				adminRouter.Post("/employee/change-permissions", h.UserHandler.ChangeUserRole)
			})
		})
	})

	return mainRouter
}
