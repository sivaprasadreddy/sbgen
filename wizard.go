package main

import (
	"fmt"
	"slices"
	"strings"
)

// The wizard's steps, in order. stepWelcome and stepComplete bracket the
// steps that make a selection.
const (
	stepWelcome = iota
	stepProject
	stepFrontend
	stepDatabase
	stepFeatures
	stepReview
	stepComplete
)

var stepNames = []string{"Welcome", "Project", "Frontend", "Database", "Features", "Review", "Complete"}

// page is a page the sidebar opens in place of the current step, which it
// returns to.
type page int

const (
	pageNone page = iota
	pageDefaults
	pageTemplates
)

// message is a line of feedback under a form, a success or an error.
type message struct {
	text  string
	error bool
}

// projectField names a field of the Project step.
type projectField int

const (
	fieldAppName projectField = iota
	fieldAppVersion
	fieldGroupID
	fieldArtifactID
	fieldPackageName
)

// wizard is the state of the window: the selection being made, the step it
// is on, and the user's templates and defaults.
type wizard struct {
	templates templateStore
	defaults  defaultsStore
	// generate generates a project and returns the archive's path.
	generate func(GeneratorConfiguration) (string, error)
	// async runs work off the UI thread, then apply on it: in a goroutine
	// and Window.Update in the app, at once in tests.
	async func(work func() (apply func()))

	step int
	page page

	// selection is what the steps chose so far.
	selection GeneratorConfiguration
	// projectConfirmed is whether the user went past the Project step, or
	// loaded a template, after which new defaults leave the project alone.
	projectConfirmed bool
	// showProjectErrors marks the empty Project fields once the user tried
	// to go on with them.
	showProjectErrors bool

	downloadedPath  string
	generationError string
	generating      bool

	templateList   []Template
	templatesError string
	// activeTemplate is the template the selection was loaded from or last
	// saved as.
	activeTemplate *Template
	templatePanel  bool
	templateName   string
	templateMsg    *message

	projectDefaults ProjectDefaults
	defaultsDraft   ProjectDefaults
	defaultsMsg     *message

	// Confirmations shown with an alert: discarding the selection, and
	// deleting deleteTarget.
	confirmingCancel bool
	confirmingDelete bool
	deleteTarget     *Template
}

// templateStore and defaultsStore are what the wizard needs of
// TemplateService and DefaultsService.
type templateStore interface {
	List() ([]Template, error)
	Save(Template) (Template, error)
	Delete(id string) error
}

type defaultsStore interface {
	Get() (ProjectDefaults, error)
	Save(ProjectDefaults) (ProjectDefaults, error)
}

func newWizard(templates templateStore, defaults defaultsStore, generate func(GeneratorConfiguration) (string, error), async func(func() func())) *wizard {
	w := &wizard{templates: templates, defaults: defaults, generate: generate, async: async}
	w.projectDefaults = builtInDefaults
	if saved, err := defaults.Get(); err == nil {
		w.projectDefaults = saved
	} else {
		w.defaultsMsg = &message{text: "Could not load defaults: " + err.Error(), error: true}
	}
	w.resetSelection()
	w.refreshTemplates()
	return w
}

// newSelection is a fresh selection: the Project step from the defaults,
// and the first choice of every other step.
func newSelection(d ProjectDefaults) GeneratorConfiguration {
	return GeneratorConfiguration{
		Project:  projectFromDefaults(d),
		Frontend: FrontendSettings{Type: FrontendNone},
		Database: DatabaseSettings{Engine: DatabasePostgreSQL, MigrationTool: MigrationFlyway, PersistenceFramework: PersistenceSpringDataJPA},
		Features: []Feature{},
		AIModels: []AIModel{},
	}
}

// projectFromDefaults is the Project step's starting values for a new
// application.
func projectFromDefaults(d ProjectDefaults) ProjectSettings {
	return ProjectSettings{
		AppName:     "demo",
		AppVersion:  d.AppVersion,
		GroupID:     d.GroupID,
		ArtifactID:  "demo",
		PackageName: deriveBasePackage(d.GroupID, "demo"),
		BuildTool:   d.BuildTool,
		TaskRunner:  d.TaskRunner,
	}
}

// deriveBasePackage is the package name the group and artifact IDs give.
func deriveBasePackage(groupID, artifactID string) string {
	group := strings.TrimSpace(groupID)
	segment := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(artifactID), "-", ""))
	switch {
	case group == "":
		return segment
	case segment == "":
		return group
	}
	return group + "." + segment
}

// projectValue returns a pointer to one of p's text fields.
func projectValue(p *ProjectSettings, f projectField) *string {
	switch f {
	case fieldAppName:
		return &p.AppName
	case fieldAppVersion:
		return &p.AppVersion
	case fieldGroupID:
		return &p.GroupID
	case fieldArtifactID:
		return &p.ArtifactID
	}
	return &p.PackageName
}

// setProjectField changes a Project field. The package name follows the
// group and artifact IDs until the user customizes it.
func (w *wizard) setProjectField(f projectField, value string) {
	p := &w.selection.Project
	derived := p.PackageName == deriveBasePackage(p.GroupID, p.ArtifactID)
	*projectValue(p, f) = value
	if derived && (f == fieldGroupID || f == fieldArtifactID) {
		p.PackageName = deriveBasePackage(p.GroupID, p.ArtifactID)
	}
	w.generationError = ""
}

// projectComplete reports whether every Project text field is filled in.
func (w *wizard) projectComplete() bool {
	for f := fieldAppName; f <= fieldPackageName; f++ {
		if strings.TrimSpace(*projectValue(&w.selection.Project, f)) == "" {
			return false
		}
	}
	return true
}

// setFrontend chooses a frontend. HTMX stays only with a server-rendered one.
func (w *wizard) setFrontend(t FrontendType) {
	w.selection.Frontend = FrontendSettings{Type: t, HTMX: isServerRendered(t) && w.selection.Frontend.HTMX}
}

func (w *wizard) toggleFeature(f Feature) {
	if i := slices.Index(w.selection.Features, f); i >= 0 {
		w.selection.Features = slices.Delete(slices.Clone(w.selection.Features), i, i+1)
		return
	}
	w.selection.Features = append(slices.Clone(w.selection.Features), f)
}

// toggleAIModel keeps the models in their canonical order.
func (w *wizard) toggleAIModel(m AIModel) {
	chosen := slices.Contains(w.selection.AIModels, m)
	models := []AIModel{}
	for _, o := range aiModelOptions {
		if (o.value == m && !chosen) || (o.value != m && slices.Contains(w.selection.AIModels, o.value)) {
			models = append(models, o.value)
		}
	}
	w.selection.AIModels = models
}

// next goes on from the current step. The Project step goes on only with
// its fields filled in.
func (w *wizard) next() {
	if w.step == stepProject {
		if !w.projectComplete() {
			w.showProjectErrors = true
			return
		}
		w.showProjectErrors = false
		w.projectConfirmed = true
	}
	if w.step < stepReview {
		w.step++
	}
}

func (w *wizard) back() {
	if w.step == stepReview {
		// A message about the selection goes stale once it can change.
		w.templatePanel = false
		w.templateMsg = nil
	}
	if w.step > stepWelcome {
		w.step--
	}
}

// getStarted begins a project. Without a template it starts from the
// defaults, not from a template looked at earlier.
func (w *wizard) getStarted() {
	if w.activeTemplate != nil {
		w.resetSelection()
	}
	w.step = stepProject
}

// resetSelection drops every choice and the loaded template.
func (w *wizard) resetSelection() {
	w.selection = newSelection(w.projectDefaults)
	w.projectConfirmed = false
	w.showProjectErrors = false
	w.activeTemplate = nil
	w.templateName = ""
	w.templatePanel = false
	w.templateMsg = nil
}

// cancel is Cancel's action once the user confirmed it.
func (w *wizard) cancel() {
	w.resetSelection()
	w.generationError = ""
	w.step = stepWelcome
}

func (w *wizard) startNew() {
	w.resetSelection()
	w.downloadedPath = ""
	w.generationError = ""
	w.generating = false
	w.step = stepWelcome
}

// applyTemplate loads a template's selection into every step, then shows
// step.
func (w *wizard) applyTemplate(t Template, step int) {
	w.selection = cloneConfiguration(t.Configuration)
	if w.selection.Features == nil {
		w.selection.Features = []Feature{}
	}
	ordered := []AIModel{}
	for _, o := range aiModelOptions {
		if slices.Contains(t.Configuration.AIModels, o.value) {
			ordered = append(ordered, o.value)
		}
	}
	w.selection.AIModels = ordered
	template := t
	w.activeTemplate = &template
	w.templateName = t.Name
	w.templatePanel = false
	w.templateMsg = nil
	w.generationError = ""
	w.projectConfirmed = true
	w.showProjectErrors = false
	w.page = pageNone
	w.step = step
}

func cloneConfiguration(c GeneratorConfiguration) GeneratorConfiguration {
	c.Features = slices.Clone(c.Features)
	c.AIModels = slices.Clone(c.AIModels)
	return c
}

// configuration is the selection as the generator and templates take it.
func (w *wizard) configuration() GeneratorConfiguration {
	c := cloneConfiguration(w.selection)
	c.Frontend.HTMX = isServerRendered(c.Frontend.Type) && c.Frontend.HTMX
	return c
}

func (w *wizard) refreshTemplates() {
	list, err := w.templates.List()
	if err != nil {
		w.templatesError = "Could not load templates: " + err.Error()
		return
	}
	w.templateList = list
	w.templatesError = ""
}

// saveTemplate saves the selection as a new template, or over the active
// one when update is true.
func (w *wizard) saveTemplate(update bool) {
	if !w.projectComplete() {
		w.showProjectErrors = true
		return
	}
	id := ""
	if update && w.activeTemplate != nil {
		id = w.activeTemplate.ID
	}
	saved, err := w.templates.Save(Template{ID: id, Name: w.templateName, Configuration: w.configuration()})
	if err != nil {
		w.templateMsg = &message{text: err.Error(), error: true}
		return
	}
	w.activeTemplate = &saved
	w.templateName = saved.Name
	verb := "Saved"
	if update {
		verb = "Updated"
	}
	w.templateMsg = &message{text: fmt.Sprintf("%s template “%s”.", verb, saved.Name)}
	w.refreshTemplates()
}

func (w *wizard) deleteTemplate(t Template) {
	if err := w.templates.Delete(t.ID); err != nil {
		w.templatesError = fmt.Sprintf("Could not delete “%s”: %v", t.Name, err)
		return
	}
	if w.activeTemplate != nil && w.activeTemplate.ID == t.ID {
		w.activeTemplate = nil
	}
	w.refreshTemplates()
}

// startGenerating generates the project off the UI thread and shows the
// Complete step, or the error, when it is done.
func (w *wizard) startGenerating() {
	if w.generating {
		return
	}
	if !w.projectComplete() {
		w.showProjectErrors = true
		return
	}
	w.generating = true
	w.generationError = ""
	configuration := w.configuration()
	w.async(func() func() {
		path, err := w.generate(configuration)
		return func() {
			w.generating = false
			if err != nil {
				w.generationError = err.Error()
				return
			}
			w.downloadedPath = path
			w.step = stepComplete
		}
	})
}

func (w *wizard) openPage(p page) {
	switch p {
	case pageTemplates:
		w.refreshTemplates()
	case pageDefaults:
		w.defaultsDraft = w.projectDefaults
		w.defaultsMsg = nil
	}
	w.page = p
}

// saveDefaults saves the Defaults page. A Project step that still shows the
// previous defaults follows the new ones; one the user changed, confirmed
// or filled from a template keeps its values.
func (w *wizard) saveDefaults() {
	saved, err := w.defaults.Save(w.defaultsDraft)
	if err != nil {
		w.defaultsMsg = &message{text: err.Error(), error: true}
		return
	}
	if !w.projectConfirmed && w.activeTemplate == nil && w.selection.Project == projectFromDefaults(w.projectDefaults) {
		w.selection.Project = projectFromDefaults(saved)
	}
	w.projectDefaults = saved
	w.defaultsDraft = saved
	w.defaultsMsg = &message{text: "Saved. New projects start with these values."}
}

// templateSummary is a one-line description of a template's main choices,
// e.g. "Maven · JTE + HTMX · PostgreSQL · Spring Data JPA · 2 features".
func templateSummary(c GeneratorConfiguration) string {
	frontend := "REST API"
	if c.Frontend.Type != FrontendNone {
		frontend = label(c.Frontend.Type)
		if c.Frontend.HTMX {
			frontend += " + HTMX"
		}
	}
	parts := []string{label(c.Project.BuildTool), frontend, label(c.Database.Engine), label(c.Database.PersistenceFramework)}
	count := func(n int, noun string) {
		switch {
		case n == 1:
			parts = append(parts, fmt.Sprintf("1 %s", noun))
		case n > 1:
			parts = append(parts, fmt.Sprintf("%d %ss", n, noun))
		}
	}
	count(len(c.Features), "feature")
	count(len(c.AIModels), "AI model")
	return strings.Join(parts, " · ")
}
