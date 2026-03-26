package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const defaultCollapseAfter = 5

var tasksTmpl = template.Must(template.New("tasks").Parse(`
{{- if not .Tasks -}}
<p class="color-subdue">No tasks</p>
{{- else -}}
<ul class="dynamic-columns list-gap-20 list-with-separator collapsible-container" data-collapse-after="{{.CollapseAfter}}">
  {{- range .Tasks}}
  <li class="flex items-center gap-15">
    <span>{{.Icon}}</span>
    <div class="grow min-width-0">
      <a class="block text-truncate color-primary-if-not-visited" href="{{.URL}}" target="_blank" rel="noreferrer">{{.Content}}</a>
      {{- if .Due}}
      <p class="size-h6 color-subdue">{{.Due}}</p>
      {{- end}}
    </div>
  </li>
  {{- end}}
</ul>
{{- end}}
`))

type templateData struct {
	Tasks         []taskView
	CollapseAfter int
}

type taskView struct {
	Icon    string
	URL     template.URL
	Content string
	Due     string
}

type Task struct {
	Content     string   `json:"content"`
	Description string   `json:"description"`
	Due         *DueDate `json:"due"`
	ID          string   `json:"id"`
	Priority    int      `json:"priority"`
	ProjectID   string   `json:"project_id"`
}

type DueDate struct {
	Date string `json:"date"`
}

type tasksResponse struct {
	Results []Task `json:"results"`
}

func main() {
	token := os.Getenv("TODOIST_API_KEY")
	if token == "" {
		log.Fatal("TODOIST_API_KEY environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		filter := r.URL.Query().Get("filter")

		collapseAfter := defaultCollapseAfter
		if v := r.URL.Query().Get("collapse_after"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				collapseAfter = n
			}
		}

		tasks, err := fetchTasks(token, filter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Widget-Title", "Todoist")
		w.Header().Set("Widget-Title-Icon", "si:todoist")
		w.Header().Set("Widget-Content-Type", "html")
		fmt.Fprint(w, renderHTML(tasks, collapseAfter))
	})

	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

const todoistBase = "https://api.todoist.com"

func taskURL(id, content string) string {
	slug := slugify(content)
	if slug == "" {
		return "https://app.todoist.com/app/task/" + id
	}
	return "https://app.todoist.com/app/task/" + slug + "-" + id
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_':
			b.WriteRune(r)
			prevDash = false
		case r == '-' || r == ' ' || r == '\t':
			if !prevDash && b.Len() > 0 {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-_")
}

func fetchTasks(token, filter string) ([]Task, error) {
	return fetchTasksFromBase(todoistBase, token, filter)
}

func fetchTasksFromBase(base, token, filter string) ([]Task, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	apiURL := base + "/api/v1/tasks"
	if filter != "" {
		apiURL = base + "/api/v1/tasks/filter"
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	if filter != "" {
		q := req.URL.Query()
		q.Set("query", filter)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("todoist API returned %d", resp.StatusCode)
	}

	var result tasksResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Results, nil
}

func priorityIcon(p int) string {
	switch p {
	case 4:
		return "🔴"
	case 3:
		return "🟠"
	case 2:
		return "🔵"
	default:
		return "⚪"
	}
}

func renderHTML(tasks []Task, collapseAfter int) string {
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Priority > tasks[j].Priority
	})

	views := make([]taskView, len(tasks))
	for i, t := range tasks {
		due := ""
		if t.Due != nil && t.Due.Date != "" {
			due = formatDue(t.Due.Date)
		}
		views[i] = taskView{
			Icon:    priorityIcon(t.Priority),
			URL:     template.URL(taskURL(t.ID, t.Content)),
			Content: t.Content,
			Due:     due,
		}
	}

	var buf bytes.Buffer
	if err := tasksTmpl.Execute(&buf, templateData{
		Tasks:         views,
		CollapseAfter: collapseAfter,
	}); err != nil {
		return `<p class="color-negative">render error</p>`
	}
	return buf.String()
}

func formatDue(date string) string {
	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	switch date {
	case today:
		return "Today"
	case tomorrow:
		return "Tomorrow"
	default:
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			return date
		}
		if t.Before(time.Now().Truncate(24 * time.Hour)) {
			return "Overdue · " + t.Format("2 Jan")
		}
		return t.Format("2 Jan")
	}
}
