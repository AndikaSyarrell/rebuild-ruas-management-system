package routes

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"

	"rms-backend/internal/config"
	"rms-backend/internal/handlers"
	"rms-backend/internal/middleware"
	"rms-backend/internal/repository"
	"rms-backend/internal/service"
	"rms-backend/internal/utils"
)

type Dependencies struct {
	Cfg		*config.Config
	Logger	*service.Logger

	DB		  	*sql.DB
	JWTManager  *utils.JWTManager
	AuthService *service.AuthService
	RDB         *redis.Client

	AccessRepo *repository.AccessRepo

	AuthHandler      *handlers.AuthHandler
	RegionHandler    *handlers.RegionHandler
	DivisionHandler  *handlers.DivisionHandler
	UnitHandler      *handlers.UnitHandler
	PpnHandler       *handlers.PpnHandler
	RoleHandler      *handlers.RoleHandler
	AccessHandler    *handlers.AccessHandler
	AdminHandler     *handlers.AdminHandler
	ClientHandler    *handlers.ClientHandler
	POHandler        *handlers.POHandler
	PRHandler        *handlers.PRHandler
	PRPaymentHandler *handlers.PRPaymentHandler
	SignatureHandler *handlers.SignatureHandler
	ResponsibleHandler *handlers.ResponsibleHandler
	PRDocumentHandler *handlers.PRDocumentHandler
	ActivityHandler  *handlers.ActivityHandler
	DocumentHandler  *handlers.DocumentHandler
	DashboardHandler *handlers.DashboardHandler
	ReportHandler    *handlers.ReportHandler
	PRCancelHandler *handlers.PRCancelHandler
}

// corsMiddleware adalah implementasi CORS minimal (tanpa dependency eksternal)
// yang membaca daftar origin yang diizinkan dari konfigurasi.
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowed[origin] || allowed["*"] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func New(d *Dependencies) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(middleware.SecurityHeaders)
	r.Use(corsMiddleware(d.Cfg.CORSAllowedOrigins))
	r.Use(middleware.RateLimit(d.RDB, d.Cfg.GlobalRateLimit, d.Cfg.GlobalRateWindow))

	auth := middleware.Auth(d.JWTManager, d.AuthService)
	requireAccess := func(slug string) func(http.Handler) http.Handler {
		return middleware.RequireAccess(d.AccessRepo, slug)
	}

	requireAny := func(slugs ...string) func(http.Handler) http.Handler {
		return middleware.RequireAnyAccess(d.AccessRepo, slugs...)
	}

	r.Handle("/docs/*", http.StripPrefix("/docs/", http.FileServer(http.Dir("./docs"))))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		utils.OK(w, "OK", nil)
	})

	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		checks := map[string]string{}
		healthy := true

		if err := d.DB.PingContext(ctx); err != nil {
			checks["database"] = "down: " + err.Error()
			healthy = false
		} else {
			checks["database"] = "ok"
		}

		if err := d.RDB.Ping(ctx).Err(); err != nil {
			checks["redis"] = "down: " + err.Error()
			healthy = false
		} else {
			checks["redis"] = "ok"
		}

		if !healthy {
			utils.JSON(w, http.StatusServiceUnavailable, false, "Service belum siap", checks)
			return
		}
		utils.JSON(w, http.StatusOK, true, "Ready", checks)
	})

	r.Route("/api/auth", func(rt chi.Router) {
		rt.Post("/login", d.AuthHandler.Login)
		rt.Post("/refresh", d.AuthHandler.Refresh)
		rt.Post("/activate", d.AuthHandler.Activate)
		rt.Post("/forgot-password", d.AuthHandler.ForgotPassword)
		rt.Post("/reset-password-code", d.AuthHandler.ResetPasswordWithCode)

		rt.Group(func(protected chi.Router) {
			protected.Use(auth)
			protected.Get("/me", d.AuthHandler.Me)
			protected.Post("/logout", d.AuthHandler.Logout)
			protected.Post("/reset-password", d.AuthHandler.ResetPassword)
		})
	})

	r.Route("/api", func(api chi.Router) {
		api.Use(auth)
		api.Use(middleware.RequestLogger(d.Logger))

		// --- Master data: Region ---
		api.Route("/regions", func(rt chi.Router) {
			rt.Get("/", d.RegionHandler.List)
			rt.Get("/select", d.RegionHandler.Select)
			rt.Get("/{id}", d.RegionHandler.Detail)
			rt.With(requireAccess("create_region")).Post("/", d.RegionHandler.Create)
			rt.With(requireAccess("edit_region")).Put("/{id}", d.RegionHandler.Update)
			rt.With(requireAccess("delete_region")).Delete("/{id}", d.RegionHandler.Delete)
		})

		// --- Master data: Division ---
		api.Route("/divisions", func(rt chi.Router) {
			rt.Get("/", d.DivisionHandler.List)
			rt.Get("/select", d.DivisionHandler.Select)
			rt.Get("/{id}", d.DivisionHandler.Detail)
			rt.With(requireAccess("create_division")).Post("/", d.DivisionHandler.Create)
			rt.With(requireAccess("edit_division")).Put("/{id}", d.DivisionHandler.Update)
			rt.With(requireAccess("delete_division")).Delete("/{id}", d.DivisionHandler.Delete)
		})

		// --- Master data: Unit ---
		api.Route("/units", func(rt chi.Router) {
			rt.Get("/", d.UnitHandler.List)
			rt.Get("/select", d.UnitHandler.Select)
			rt.Get("/{id}", d.UnitHandler.Detail)
			rt.With(requireAccess("create_unit")).Post("/", d.UnitHandler.Create)
			rt.With(requireAccess("edit_unit")).Put("/{id}", d.UnitHandler.Update)
			rt.With(requireAccess("delete_unit")).Delete("/{id}", d.UnitHandler.Delete)
		})

		// --- Master data: PPN ---
		api.Route("/ppn", func(rt chi.Router) {
			rt.Get("/", d.PpnHandler.List)
			rt.Get("/current", d.PpnHandler.Current)
			rt.Get("/{id}", d.PpnHandler.Detail) // baru
			rt.With(requireAccess("edit_ppn")).Put("/{id}", d.PpnHandler.Update)
			rt.With(requireAccess("edit_ppn")).Post("/", d.PpnHandler.Create)
		})

		// --- RBAC: Role & Access ---
		// PENTING: modul ini mengatur akses itu sendiri — wajib digating,
		// jangan biarkan sembarang admin login bisa buat/ubah role & akses.
		api.Route("/roles", func(rt chi.Router) {
			rt.Get("/", d.RoleHandler.List)
			rt.Get("/select", d.RoleHandler.Select)
			rt.Get("/{id}", d.RoleHandler.Detail)
			rt.With(requireAccess("create_role")).Post("/", d.RoleHandler.Create)
			rt.With(requireAccess("edit_role")).Put("/{id}", d.RoleHandler.Update)
			rt.With(requireAccess("delete_role")).Delete("/{id}", d.RoleHandler.Delete)
		})
		api.Route("/access", func(rt chi.Router) {
			rt.Get("/mine", d.AccessHandler.Mine)
			rt.Get("/", d.AccessHandler.List)
			rt.Get("/module/{module}", d.AccessHandler.ListByModule)
			rt.Get("/{id}", d.AccessHandler.Detail)
			rt.With(requireAccess("create_access")).Post("/", d.AccessHandler.Create)
			rt.With(requireAccess("edit_access")).Put("/{id}", d.AccessHandler.Update)
			rt.With(requireAccess("delete_access")).Delete("/{id}", d.AccessHandler.Delete)
		})

		// --- Admin (internal users) ---
		api.Route("/admins", func(rt chi.Router) {
			rt.Get("/", d.AdminHandler.List)
			rt.Get("/pic", d.AdminHandler.ListPIC)
			rt.Get("/pic-client", d.AdminHandler.ListPICClient)
			rt.Get("/checkers", d.AdminHandler.ListCheckers)
			rt.Get("/{id}", d.AdminHandler.Detail)
			rt.Get("/me/signature", d.SignatureHandler.Mine)
			rt.Post("/me/signature", d.SignatureHandler.Upload)
			rt.With(requireAccess("create_new_admin")).Post("/", d.AdminHandler.Create)
			rt.With(requireAccess("edit_other_admin")).Put("/{id}", d.AdminHandler.Update)
			rt.With(requireAccess("edit_pic")).Post("/{id}/image", d.AdminHandler.UploadImage)
			rt.With(requireAccess("resend_admin_activation")).Post("/{id}/resend-activation", d.AdminHandler.ResendActivation)
			rt.With(requireAccess("deactivate_other_admin")).Post("/{id}/deactivate", d.AdminHandler.Deactivate)
			rt.With(requireAccess("deactivate_other_admin")).Post("/{id}/activate", d.AdminHandler.ActivateExisting)
			rt.With(requireAccess("delete_admin")).Delete("/{id}", d.AdminHandler.Delete)
		})

		// --- Client ---
		api.Route("/clients", func(rt chi.Router) {
			rt.Get("/", d.ClientHandler.List)
			rt.Get("/select", d.ClientHandler.Select)
			rt.Get("/{id}", d.ClientHandler.Detail)
			rt.With(requireAccess("create_client")).Post("/", d.ClientHandler.Create)
			rt.With(requireAccess("edit_client")).Put("/{id}", d.ClientHandler.Update)
			rt.With(requireAccess("delete_client")).Delete("/{id}", d.ClientHandler.Delete)
			rt.With(requireAccess("change_stat_active")).Post("/{id}/activate", d.ClientHandler.Activate)
			rt.With(requireAccess("change_stat_active")).Post("/{id}/deactivate", d.ClientHandler.Deactivate)
		})

		// --- Purchase Order ---
		api.Route("/po", func(rt chi.Router) {
			rt.Get("/", d.POHandler.List)
			rt.Get("/total", d.POHandler.Total)
			rt.With(requireAccess("export_po")).Get("/export", d.POHandler.Export)
			rt.Get("/{id}", d.POHandler.Detail)
			rt.Get("/{id}/linked-quotations", d.POHandler.LinkedQuotations)
			rt.Get("/{id}/linked-quotations/{quotationId}/prs", d.POHandler.LinkedQuotationPRs)
			rt.With(requireAccess("create_po")).Post("/", d.POHandler.Create)
			rt.With(requireAccess("edit_po")).Put("/{id}", d.POHandler.Update)
			rt.With(requireAccess("delete_po")).Delete("/{id}", d.POHandler.Delete)

			rt.With(requireAccess("change_stat_po")).Post("/{id}/status", d.POHandler.ChangeStatus)
			rt.With(requireAccess("change_stat_paid")).Post("/{id}/paid", d.POHandler.ChangePaid)
			rt.With(requireAccess("update_invoice")).Post("/{id}/invoice", d.POHandler.UpdateInvoice)
			rt.With(requireAccess("update_po_notes")).Post("/{id}/notes", d.POHandler.UpdateNotes)

			rt.Get("/{id}/items", d.POHandler.ListItems)
			rt.With(requireAccess("edit_po")).Post("/{id}/items", d.POHandler.AddItem)
			rt.With(requireAccess("edit_po")).Put("/items/{itemId}", d.POHandler.UpdateItem)
			rt.With(requireAccess("edit_po")).Delete("/items/{itemId}", d.POHandler.DeleteItem)

			rt.With(requireAccess("link_pr_po")).Get("/{id}/quotation-candidates", d.POHandler.QuotationCandidates)
			rt.With(requireAccess("link_pr_po")).Post("/{id}/link-quotation", d.POHandler.LinkQuotation)
			rt.With(requireAccess("unlink_pr_po")).Post("/{id}/unlink-quotation", d.POHandler.UnlinkQuotation)

			rt.With(requireAccess("edit_po")).Post("/{id}/quot-hint", d.POHandler.UpdateQuotHint)

			rt.Get("/{id}/activities", d.ActivityHandler.ListByPO)
			rt.With(requireAccess("create_notes")).Post("/{id}/notes-activity", d.ActivityHandler.AddNote)

			rt.Get("/{id}/documents", d.DocumentHandler.ListByPO)
			rt.Get("/{id}/documents/{docId}/download", d.DocumentHandler.Download)
			rt.With(requireAccess("upload_document")).Post("/{id}/documents", d.DocumentHandler.Upload)
			rt.With(requireAccess("upload_document")).Delete("/{id}/documents/{docId}", d.DocumentHandler.Delete)
		})

		// --- Purchase Request ---
		api.Route("/pr", func(rt chi.Router) {
			rt.Get("/", d.PRHandler.List)
			rt.Get("/{id}", d.PRHandler.Detail)
			rt.With(requireAccess("export_pr")).Get("/{id}/export", d.PRHandler.ExportRFP)
			rt.With(requireAccess("create_pr")).Post("/", d.PRHandler.Create)
			rt.With(requireAccess("edit_pr")).Put("/{id}", d.PRHandler.Update)
			rt.Post("/{id}/cancel", d.PRCancelHandler.Cancel)
			rt.Post("/{id}/cancel-request", d.PRCancelHandler.CreateRequest)
			rt.Get("/{id}/cancel-requests", d.PRCancelHandler.ListByPR)

			rt.With(requireAccess("finance")).Get("/cancel-requests", d.PRCancelHandler.ListForFinance)
			rt.Get("/cancel-requests/{cancelId}/document", d.PRCancelHandler.Document)
			rt.With(requireAccess("finance")).Post("/cancel-requests/{cancelId}/approve", d.PRCancelHandler.Approve)
			rt.With(requireAccess("finance")).Post("/cancel-requests/{cancelId}/reject", d.PRCancelHandler.Reject)
			
			rt.Get("/{id}/documents", d.PRDocumentHandler.ListByPR)
			rt.Post("/{id}/documents", d.PRDocumentHandler.Upload)
			rt.Delete("/{id}/documents/{docId}", d.PRDocumentHandler.Delete)
			rt.Get("/{id}/documents/{docId}/download", d.PRDocumentHandler.Download) // BARU
			rt.Get("/{id}/documents/{docId}/preview", d.PRDocumentHandler.Preview)   // BARU
			rt.With(requireAccess("export_pr_payment_report")).Get("/payments/export", d.PRPaymentHandler.Export)
			
			rt.Get("/{id}/history", d.PRHandler.History)
			rt.Get("/{id}/approvals", d.PRHandler.ListApprovals)
			rt.With(requireAccess("submit_pr")).Post("/{id}/submit", d.PRHandler.Submit)
			rt.With(requireAccess("submit_pr")).Post("/bulk-submit", d.PRHandler.BulkSubmit)
			rt.With(requireAccess("create_pr_payment"), requireAccess("finance")).Post("/{id}/payments", d.PRPaymentHandler.Create)
			rt.With(requireAccess("finance")).Post("/payments/{paymentId}/priority-date", d.PRPaymentHandler.SetPriorityDate)
			
			rt.Get("/my-approvals", d.PRHandler.MyTurn)
			rt.Get("/my-decisions", d.PRHandler.MyDecisions)
			rt.With(requireAccess("create_pr")).Get("/quotations", d.PRHandler.SearchQuotations)
			rt.With(requireAny("checker", "director", "finance")).Post("/approvals/{approvalId}/approve", d.PRHandler.ApproveApproval)
			rt.With(requireAny("checker", "director", "finance")).Post("/approvals/bulk-decide", d.PRHandler.BulkDecide)
			rt.With(requireAny("checker", "director", "finance")).Post("/approvals/{approvalId}/reject", d.PRHandler.RejectApproval)
			rt.With(requireAccess("checker")).Post("/approvals/{approvalId}/request-revision", d.PRHandler.RequestRevision)
			
			rt.Get("/{id}/comments", d.PRHandler.ListComments)
			rt.Post("/{id}/comments", d.PRHandler.AddComment)
			
			rt.With(requireAccess("request_revision_pr")).Post("/approvals/{approvalId}/request-revision", d.PRHandler.RequestRevision)
			rt.With(requireAccess("update_pr_amounts")).Post("/{id}/amounts", d.PRHandler.UpdateAmounts)
			
			rt.Get("/{id}/payments", d.PRPaymentHandler.ListByPR)
			rt.Get("/{id}/payment-chain", d.PRPaymentHandler.PaymentChain)
			rt.With(requireAccess("create_pr_payment")).Post("/{id}/payments", d.PRPaymentHandler.Create)
			rt.With(requireAccess("confirm_pr_payment")).Post("/payments/{paymentId}/confirm", d.PRPaymentHandler.Confirm)
			rt.With(requireAccess("confirm_pr_payment")).Post("/payments/{paymentId}/cancel", d.PRPaymentHandler.Cancel)
		})

		api.Route("/responsibles", func(rt chi.Router) {
			rt.Get("/", d.ResponsibleHandler.List)
			rt.Get("/select", d.ResponsibleHandler.Select)
			rt.Get("/{id}", d.ResponsibleHandler.Detail)
			rt.With(requireAccess("create_responsible")).Post("/", d.ResponsibleHandler.Create)
			rt.With(requireAccess("edit_responsible")).Put("/{id}", d.ResponsibleHandler.Update)
			rt.With(requireAccess("delete_responsible")).Delete("/{id}", d.ResponsibleHandler.Delete)
			rt.With(requireAccess("change_stat_responsible")).Post("/{id}/activate", d.ResponsibleHandler.Activate)
			rt.With(requireAccess("change_stat_responsible")).Post("/{id}/deactivate", d.ResponsibleHandler.Deactivate)
		})

		api.Get("/activities/dashboard", d.ActivityHandler.Dashboard)
		api.Get("/dashboard", d.DashboardHandler.Summary)

		api.Route("/reports", func(rt chi.Router) {
			rt.Use(requireAccess("view_report_table"))
			rt.Get("/years", d.ReportHandler.Years)
			rt.Get("/chart", d.ReportHandler.Chart)
			rt.Get("/bars", d.ReportHandler.Bars)
			rt.Get("/stats", d.ReportHandler.Stats)
		})
	})

	return r
}
