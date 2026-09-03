package app

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/olivecarney/splitkit-go/internal/balances"
	"github.com/olivecarney/splitkit-go/internal/expenses"
	"github.com/olivecarney/splitkit-go/internal/groups"
	"github.com/olivecarney/splitkit-go/internal/models"
	"github.com/olivecarney/splitkit-go/internal/settlements"
)

type DataStore interface {
	CreateGroup(ctx context.Context, name string, createdBy models.User) (models.Group, error)
	ListGroups(ctx context.Context) ([]models.Group, error)
	GetGroup(ctx context.Context, id string) (models.Group, error)
	DeleteGroup(ctx context.Context, id string) error
	AddMember(ctx context.Context, groupID string, displayName string) (models.Member, error)
	RemoveMember(ctx context.Context, groupID string, memberID string) error
	ListMembers(ctx context.Context, groupID string) ([]models.Member, error)
	CreateExpense(ctx context.Context, input models.CreateExpenseInput) (models.Expense, error)
	ListExpenses(ctx context.Context, groupID string) ([]models.Expense, error)
	MarkSettlementPaid(ctx context.Context, input models.Settlement) (models.Settlement, error)
	ListSettlements(ctx context.Context, groupID string) ([]models.Settlement, error)
}

type Config struct {
	Addr          string
	Store         DataStore
	DevUser       models.User
	StaticDir     string
	TemplatesGlob string
}

type Server struct {
	addr        string
	mux         *http.ServeMux
	templates   *template.Template
	devUser     models.User
	groups      groups.Service
	expenses    expenses.Service
	balances    balances.Service
	settlements settlements.Service
}

func NewServer(cfg Config) *Server {
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"money":       formatMoney,
		"signedMoney": formatSignedMoney,
	}).ParseGlob(cfg.TemplatesGlob))

	s := &Server{
		addr:      cfg.Addr,
		mux:       http.NewServeMux(),
		templates: tmpl,
		devUser:   cfg.DevUser,
		groups: groups.Service{
			Store: cfg.Store,
		},
		expenses: expenses.Service{
			Store: cfg.Store,
		},
		balances: balances.Service{
			Store: cfg.Store,
		},
		settlements: settlements.Service{
			Store: cfg.Store,
		},
	}

	s.routes(cfg.StaticDir)
	return s
}

func (s *Server) ListenAndServe() error {
	return http.ListenAndServe(s.addr, s.mux)
}

func (s *Server) routes(staticDir string) {
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	s.mux.HandleFunc("GET /", s.home)
	s.mux.HandleFunc("GET /groups", s.groupList)
	s.mux.HandleFunc("GET /groups/new", s.groupNew)
	s.mux.HandleFunc("POST /groups", s.groupCreate)
	s.mux.HandleFunc("GET /groups/", s.groupShow)
	s.mux.HandleFunc("POST /groups/", s.groupAction)
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	s.render(w, "home", pageData{Title: "SplitKit"})
}

func (s *Server) groupList(w http.ResponseWriter, r *http.Request) {
	list, err := s.groups.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, "groups", pageData{Title: "Groups", Groups: list})
}

func (s *Server) groupNew(w http.ResponseWriter, r *http.Request) {
	s.render(w, "group_new", pageData{Title: "New group"})
}

func (s *Server) groupCreate(w http.ResponseWriter, r *http.Request) {
	group, err := s.groups.Create(r.Context(), strings.TrimSpace(r.FormValue("name")), s.devUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/groups/"+group.ID, http.StatusSeeOther)
}

func (s *Server) groupShow(w http.ResponseWriter, r *http.Request) {
	groupID := pathID(r.URL.Path, "/groups/")
	dashboard, err := s.loadDashboard(r.Context(), groupID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	dashboard.FocusMember = r.URL.Query().Get("focus") == "member"
	s.render(w, "group_show", dashboard)
}

func (s *Server) groupAction(w http.ResponseWriter, r *http.Request) {
	groupID, action := pathAction(r.URL.Path, "/groups/")
	redirectTo := "/groups/" + groupID
	switch action {
	case "delete":
		if err := s.groups.Delete(r.Context(), groupID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/groups", http.StatusSeeOther)
		return
	case "members":
		_, err := s.groups.AddMember(r.Context(), groupID, strings.TrimSpace(r.FormValue("display_name")))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		redirectTo += "?focus=member#members"
	case "members/remove":
		if err := s.groups.RemoveMember(r.Context(), groupID, r.FormValue("member_id")); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		redirectTo += "#members"
	case "expenses":
		_, err := s.expenses.Create(r.Context(), models.CreateExpenseInput{
			GroupID:     groupID,
			PaidByID:    r.FormValue("paid_by_id"),
			Description: strings.TrimSpace(r.FormValue("description")),
			Amount:      strings.TrimSpace(r.FormValue("amount")),
			SplitWith:   r.Form["split_with"],
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		redirectTo += "#expenses"
	case "settlements":
		_, err := s.settlements.MarkPaid(r.Context(), models.Settlement{
			GroupID:     groupID,
			FromUserID:  r.FormValue("from_user_id"),
			ToUserID:    r.FormValue("to_user_id"),
			AmountCents: centsFromForm(r.FormValue("amount_cents")),
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		redirectTo += "#settlements"
	default:
		http.NotFound(w, r)
		return
	}

	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}

func (s *Server) loadDashboard(ctx context.Context, groupID string) (pageData, error) {
	group, err := s.groups.Get(ctx, groupID)
	if err != nil {
		return pageData{}, err
	}
	members, err := s.groups.Members(ctx, groupID)
	if err != nil {
		return pageData{}, err
	}
	expenseList, err := s.expenses.List(ctx, groupID)
	if err != nil {
		return pageData{}, err
	}
	balanceList, suggestions, err := s.balances.ForGroup(ctx, groupID)
	if err != nil {
		return pageData{}, err
	}
	paid, err := s.settlements.List(ctx, groupID)
	if err != nil {
		return pageData{}, err
	}
	return pageData{
		Title:       group.Name,
		Group:       group,
		Members:     members,
		Expenses:    expenseList,
		Balances:    balanceList,
		Suggestions: suggestions,
		Settlements: paid,
	}, nil
}

func (s *Server) render(w http.ResponseWriter, name string, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func pathID(path string, prefix string) string {
	rest := strings.TrimPrefix(path, prefix)
	return strings.Trim(rest, "/")
}

func pathAction(path string, prefix string) (string, string) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(path, prefix), "/"), "/")
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], strings.Join(parts[1:], "/")
}

func centsFromForm(value string) int64 {
	var cents int64
	for _, char := range value {
		if char >= '0' && char <= '9' {
			cents = cents*10 + int64(char-'0')
		}
	}
	return cents
}

func formatMoney(cents int64) string {
	if cents < 0 {
		cents = -cents
	}
	return fmt.Sprintf("£%d.%02d", cents/100, cents%100)
}

func formatSignedMoney(cents int64) string {
	prefix := "+"
	if cents < 0 {
		prefix = "-"
	}
	return prefix + formatMoney(cents)
}
