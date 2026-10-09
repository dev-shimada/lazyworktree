// Package tui implements the lazyworktree terminal UI: a list of git worktrees
// with create, delete, lock, and prune actions, plus read-only Issues/PR tabs
// and a herdr hand-off action.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dev-shimada/lazyworktree/internal/git"
	"github.com/dev-shimada/lazyworktree/internal/github"
	"github.com/dev-shimada/lazyworktree/internal/herdr"
)

type view int

const (
	viewList view = iota
	viewCreate
	viewConfirm
	viewRename
)

type tab int

const (
	tabWorktrees tab = iota
	tabIssues
	tabPRs
	tabCount
)

// Model is the root bubbletea model for lazyworktree.
type Model struct {
	repoDir  string
	repoRoot string

	state     view
	activeTab tab

	worktreeList list.Model
	issueList    list.Model
	prList       list.Model

	worktrees    []git.Worktree
	herdrLabels  map[string]string     // worktree path -> live herdr workspace label
	worktreeRefs map[string]github.Ref // worktree path -> tracked PR/issue, in any state

	issuesLoaded bool
	// issueSources holds any configured "owner/repo" alternates first, then
	// "" (the current repo) last — always present, so it's always reachable.
	issueSources   []string
	issueSourceIdx int
	issuesMine     bool // true: only issues assigned to the authenticated user
	prsLoaded      bool
	prsMine        bool // true: only PRs assigned to the authenticated user

	// issueQuery / prQuery are the active server-side searches ("" = none).
	issueQuery string
	prQuery    string
	searching  bool
	search     textinput.Model

	form    createForm
	confirm confirmDelete
	rename  renameForm

	herdrMode bool

	status    string
	statusErr bool

	width, height int

	// SelectedPath is set when the user picks a worktree to switch into.
	// Callers should print it to stdout after the program exits so a shell
	// wrapper can `cd` into it.
	SelectedPath string

	quitting bool
}

// New builds a Model rooted at the git repository containing dir.
func New(dir string) (Model, error) {
	root, err := git.RepoRoot(dir)
	if err != nil {
		return Model{}, fmt.Errorf("not inside a git repository: %w", err)
	}
	// Anchor to the main worktree, not whichever worktree dir happens to be
	// inside: new-worktree path conventions and herdr's worktree open/create
	// (which reject a linked worktree's own path outright) both need the
	// repo's canonical root, not "wherever lazyworktree was launched from."
	if mainRoot, err := git.MainWorktreeRoot(dir); err == nil && mainRoot != "" {
		root = mainRoot
	}

	newList := func() list.Model {
		l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
		l.SetShowTitle(false)
		l.SetShowStatusBar(false)
		l.SetFilteringEnabled(true)
		l.DisableQuitKeybindings()
		return l
	}
	// Issues/PRs are searched server-side (so matches aren't limited to the
	// first page loaded); bubbles' local filter would only see that page.
	newRemoteList := func() list.Model {
		l := newList()
		l.SetFilteringEnabled(false)
		return l
	}

	return Model{
		repoDir:      dir,
		repoRoot:     root,
		state:        viewList,
		worktreeList: newList(),
		issueList:    newRemoteList(),
		prList:       newRemoteList(),
		herdrMode:    herdr.InHerdr(),
		issuesMine:   true,
		prsMine:      true,
	}, nil
}

func (m Model) Init() tea.Cmd {
	return m.reloadWorktreesCmd()
}

// reloadWorktreesCmd reloads the worktree list, and in herdr mode also
// re-resolves each worktree's live herdr workspace label.
func (m Model) reloadWorktreesCmd() tea.Cmd {
	if m.herdrMode {
		return tea.Batch(loadWorktreesCmd(m.repoRoot), loadHerdrLabelsCmd(m.repoRoot))
	}
	return loadWorktreesCmd(m.repoRoot)
}

func (m *Model) rebuildWorktreeItems() {
	items := make([]list.Item, len(m.worktrees))
	for i, w := range m.worktrees {
		item := worktreeItem{wt: w}
		if ref, ok := m.worktreeRefs[w.Path]; ok {
			r := ref
			item.ref = &r
		}
		if label, ok := m.herdrLabels[w.Path]; ok {
			item.herdrLabel = label
		}
		items[i] = item
	}
	m.worktreeList.SetItems(items)
}

// currentIssueSource returns the "owner/repo" the Issues tab is currently
// backed by, or "" for the current repo (resolved via cwd).
func (m Model) currentIssueSource() string {
	if m.issueSourceIdx < len(m.issueSources) {
		return m.issueSources[m.issueSourceIdx]
	}
	return ""
}

func (m *Model) currentListPtr() *list.Model {
	switch m.activeTab {
	case tabIssues:
		return &m.issueList
	case tabPRs:
		return &m.prList
	default:
		return &m.worktreeList
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		h := msg.Height - 4
		m.worktreeList.SetSize(msg.Width, h)
		m.issueList.SetSize(msg.Width, h)
		m.prList.SetSize(msg.Width, h)
		return m, nil

	case worktreesLoadedMsg:
		m.worktrees = msg.worktrees
		m.rebuildWorktreeItems()
		return m, loadWorktreeRefsCmd(m.repoRoot, m.worktrees)

	case herdrLabelsLoadedMsg:
		m.herdrLabels = msg.labelByPath
		m.rebuildWorktreeItems()
		return m, nil

	case worktreeRefsLoadedMsg:
		m.worktreeRefs = msg.refByPath
		m.rebuildWorktreeItems()
		return m, nil

	case issuesLoadedMsg:
		m.issuesLoaded = true
		m.issueSources = msg.sources
		m.issueSourceIdx = msg.sourceIdx
		m.issuesMine = msg.mine
		m.issueQuery = msg.query
		items := make([]list.Item, len(msg.issues))
		for i, is := range msg.issues {
			items[i] = issueItem{issue: is}
		}
		m.issueList.SetItems(items)
		m.status = ""
		return m, nil

	case prsLoadedMsg:
		m.prsLoaded = true
		m.prsMine = msg.mine
		m.prQuery = msg.query
		items := make([]list.Item, len(msg.prs))
		for i, p := range msg.prs {
			items[i] = prItem{pr: p}
		}
		m.prList.SetItems(items)
		m.status = ""
		return m, nil

	case errMsg:
		m.status = msg.err.Error()
		m.statusErr = true
		return m, nil

	case actionDoneMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.statusErr = true
			return m, m.reloadWorktreesCmd()
		}
		m.status = msg.status
		m.statusErr = false
		if msg.quit {
			if msg.cdPath != "" {
				m.SelectedPath = msg.cdPath
			}
			m.quitting = true
			return m, tea.Quit
		}
		return m, m.reloadWorktreesCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		return m.handleSearchKey(msg)
	}
	switch m.state {
	case viewCreate:
		return m.handleCreateKey(msg)
	case viewConfirm:
		return m.handleConfirmKey(msg)
	case viewRename:
		return m.handleRenameKey(msg)
	default:
		return m.handleListKey(msg)
	}
}

func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cur := m.currentListPtr()
	if cur.FilterState() == list.Filtering {
		var cmd tea.Cmd
		*cur, cmd = cur.Update(msg)
		return m, cmd
	}

	if msg.String() == "esc" {
		switch {
		case m.activeTab == tabIssues && m.issueQuery != "":
			m.status = "loading issues..."
			return m, loadIssuesCmd(m.repoRoot, m.issueSourceIdx, m.issuesMine, "")
		case m.activeTab == tabPRs && m.prQuery != "":
			m.status = "loading pull requests..."
			return m, loadPRsCmd(m.repoRoot, m.prsMine, "")
		}
	}

	switch {
	case key.Matches(msg, listKeys.Quit):
		m.quitting = true
		return m, tea.Quit

	case key.Matches(msg, listKeys.CycleTab):
		m.activeTab = (m.activeTab + 1) % tabCount
		m.status = ""
		m.statusErr = false
		switch m.activeTab {
		case tabIssues:
			if !m.issuesLoaded {
				m.status = "loading issues..."
				return m, loadIssuesCmd(m.repoRoot, 0, m.issuesMine, m.issueQuery)
			}
		case tabPRs:
			if !m.prsLoaded {
				m.status = "loading pull requests..."
				return m, loadPRsCmd(m.repoRoot, m.prsMine, m.prQuery)
			}
		}
		return m, nil

	case key.Matches(msg, listKeys.Refresh):
		m.status = ""
		switch m.activeTab {
		case tabIssues:
			return m, loadIssuesCmd(m.repoRoot, m.issueSourceIdx, m.issuesMine, m.issueQuery)
		case tabPRs:
			return m, loadPRsCmd(m.repoRoot, m.prsMine, m.prQuery)
		default:
			return m, m.reloadWorktreesCmd()
		}
	}

	if m.activeTab != tabWorktrees {
		return m.handleReadonlyKey(msg)
	}
	return m.handleWorktreeKey(msg)
}

func (m Model) handleWorktreeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, listKeys.New):
		m.form = newCreateForm(m.repoRoot, m.herdrMode)
		m.state = viewCreate
		return m, nil

	case key.Matches(msg, listKeys.Delete):
		if item, ok := m.worktreeList.SelectedItem().(worktreeItem); ok {
			m.confirm = confirmDelete{target: item.wt}
			m.state = viewConfirm
		}
		return m, nil

	case key.Matches(msg, listKeys.Prune):
		m.status = "pruning..."
		m.statusErr = false
		return m, pruneWorktreesCmd(m.repoRoot)

	case key.Matches(msg, listKeys.Lock):
		if item, ok := m.worktreeList.SelectedItem().(worktreeItem); ok {
			return m, toggleLockCmd(m.repoRoot, item.wt.Path, !item.wt.Locked)
		}
		return m, nil

	case key.Matches(msg, listKeys.Open):
		item, ok := m.worktreeList.SelectedItem().(worktreeItem)
		if !ok {
			return m, nil
		}
		if item.wt.Branch == "" {
			m.status = "no branch to look up on GitHub"
			m.statusErr = true
			return m, nil
		}
		m.status = "opening in browser..."
		m.statusErr = false
		return m, openWorktreeBrowserCmd(m.repoRoot, item.wt.Branch, item.ref)

	case key.Matches(msg, listKeys.Cd):
		if item, ok := m.worktreeList.SelectedItem().(worktreeItem); ok {
			m.SelectedPath = item.wt.Path
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case key.Matches(msg, listKeys.Rename):
		if m.herdrMode {
			m.rename = newRenameForm(true, "")
			m.state = viewRename
			return m, nil
		}
		if item, ok := m.worktreeList.SelectedItem().(worktreeItem); ok {
			m.rename = newRenameForm(false, item.wt.Branch)
			m.state = viewRename
		}
		return m, nil

	case key.Matches(msg, listKeys.Select):
		item, ok := m.worktreeList.SelectedItem().(worktreeItem)
		if !ok {
			return m, nil
		}
		if !herdr.Available() {
			m.status = "herdr not found in PATH"
			m.statusErr = true
			return m, nil
		}
		m.status = "opening in herdr..."
		m.statusErr = false
		return m, openInHerdrCmd(m.repoRoot, item.wt.Path)
	}

	var cmd tea.Cmd
	m.worktreeList, cmd = m.worktreeList.Update(msg)
	return m, cmd
}

func (m Model) handleReadonlyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, listKeys.Open) {
		switch m.activeTab {
		case tabIssues:
			if item, ok := m.issueList.SelectedItem().(issueItem); ok {
				m.status = "opening in browser..."
				m.statusErr = false
				return m, openIssueBrowserCmd(m.repoRoot, m.currentIssueSource(), item.issue.Number)
			}
		case tabPRs:
			if item, ok := m.prList.SelectedItem().(prItem); ok {
				m.status = "opening in browser..."
				m.statusErr = false
				return m, openPRBrowserCmd(m.repoRoot, item.pr.Number)
			}
		}
		return m, nil
	}

	if key.Matches(msg, listKeys.Search) {
		m.search = textinput.New()
		m.search.Prompt = "search: "
		m.search.Placeholder = "GitHub search across all open items (empty = clear)"
		m.search.CharLimit = 200
		m.search.Width = 60
		if m.activeTab == tabIssues {
			m.search.SetValue(m.issueQuery)
		} else {
			m.search.SetValue(m.prQuery)
		}
		m.search.CursorEnd()
		m.search.Focus()
		m.searching = true
		return m, textinput.Blink
	}

	if key.Matches(msg, listKeys.SwitchSource) && m.activeTab == tabIssues {
		if len(m.issueSources) <= 1 {
			m.status = "no additional issue repos configured"
			m.statusErr = false
			return m, nil
		}
		newIdx := (m.issueSourceIdx + 1) % len(m.issueSources)
		m.status = "loading issues..."
		m.statusErr = false
		return m, loadIssuesCmd(m.repoRoot, newIdx, m.issuesMine, m.issueQuery)
	}

	if key.Matches(msg, listKeys.ToggleMine) {
		switch m.activeTab {
		case tabIssues:
			m.status = "loading issues..."
			m.statusErr = false
			return m, loadIssuesCmd(m.repoRoot, m.issueSourceIdx, !m.issuesMine, m.issueQuery)
		case tabPRs:
			m.status = "loading pull requests..."
			m.statusErr = false
			return m, loadPRsCmd(m.repoRoot, !m.prsMine, m.prQuery)
		}
		return m, nil
	}

	if key.Matches(msg, listKeys.Select) {
		switch m.activeTab {
		case tabIssues:
			if item, ok := m.issueList.SelectedItem().(issueItem); ok {
				m.status = "checking out issue..."
				m.statusErr = false
				return m, checkoutIssueWorktreeCmd(m.repoRoot, m.worktrees, item.issue, m.herdrMode, herdr.Available())
			}
		case tabPRs:
			if item, ok := m.prList.SelectedItem().(prItem); ok {
				m.status = "checking out PR..."
				m.statusErr = false
				return m, checkoutPRWorktreeCmd(m.repoRoot, m.worktrees, item.pr, m.herdrMode, herdr.Available())
			}
		}
		return m, nil
	}

	cur := m.currentListPtr()
	var cmd tea.Cmd
	*cur, cmd = cur.Update(msg)
	return m, cmd
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		return m, nil
	case "enter":
		m.searching = false
		query := strings.TrimSpace(m.search.Value())
		m.statusErr = false
		switch m.activeTab {
		case tabIssues:
			m.status = "searching issues..."
			return m, loadIssuesCmd(m.repoRoot, m.issueSourceIdx, m.issuesMine, query)
		case tabPRs:
			m.status = "searching pull requests..."
			return m, loadPRsCmd(m.repoRoot, m.prsMine, query)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	return m, cmd
}

func (m Model) handleCreateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = viewList
		return m, nil
	case "enter":
		if m.form.focus == fieldBranch {
			m.form.focus = fieldPath
			m.form.syncFocus()
			return m, nil
		}
		branch := m.form.branch.Value()
		path := m.form.path.Value()
		if branch == "" || path == "" {
			m.form.err = "branch and path are required"
			return m, nil
		}
		dest, opts := m.form.result()
		openHerdr := m.form.openHerdr && m.form.herdrAvailable
		m.state = viewList
		m.status = "creating worktree..."
		m.statusErr = false
		return m, addWorktreeCmd(m.repoRoot, dest, opts, openHerdr)
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		path := m.confirm.target.Path
		force := m.confirm.force
		m.state = viewList
		m.status = "removing worktree..."
		m.statusErr = false
		return m, removeWorktreeCmd(m.repoRoot, path, force)
	case "f":
		m.confirm.force = !m.confirm.force
		return m, nil
	case "n", "esc":
		m.state = viewList
		return m, nil
	}
	return m, nil
}

func (m Model) handleRenameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = viewList
		return m, nil
	case "enter":
		newName := strings.TrimSpace(m.rename.input.Value())
		if newName == "" {
			return m, nil
		}
		m.state = viewList
		m.status = "renaming..."
		m.statusErr = false
		if m.rename.herdr {
			return m, renameHerdrWorkspaceCmd(newName)
		}
		return m, renameBranchCmd(m.repoRoot, m.rename.target, newName)
	}

	var cmd tea.Cmd
	m.rename, cmd = m.rename.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	switch m.state {
	case viewCreate:
		return m.form.View()
	case viewConfirm:
		return m.confirm.View()
	case viewRename:
		return m.rename.View()
	default:
		return m.listView()
	}
}

func (m Model) listView() string {
	issueParts := []string{"all"}
	if m.issuesMine {
		issueParts = []string{"mine"}
	}
	if len(m.issueSources) > 1 {
		src := m.currentIssueSource()
		if src == "" {
			src = "this repo"
		}
		issueParts = append(issueParts, src)
	}
	if m.issueQuery != "" {
		issueParts = append(issueParts, fmt.Sprintf("search: %q", m.issueQuery))
	}
	issuesLabel := fmt.Sprintf("Issues [%s]", strings.Join(issueParts, ", "))

	prParts := []string{"all"}
	if m.prsMine {
		prParts = []string{"mine"}
	}
	if m.prQuery != "" {
		prParts = append(prParts, fmt.Sprintf("search: %q", m.prQuery))
	}
	prsLabel := fmt.Sprintf("Pull Requests [%s]", strings.Join(prParts, ", "))

	tabLabels := []struct {
		tab   tab
		label string
	}{
		{tabWorktrees, "Worktrees"},
		{tabIssues, issuesLabel},
		{tabPRs, prsLabel},
	}
	tabBar := ""
	for _, t := range tabLabels {
		if t.tab == m.activeTab {
			tabBar += activeTabStyle.Render(t.label)
		} else {
			tabBar += inactiveTabStyle.Render(t.label)
		}
	}

	statusLine := ""
	if m.status != "" {
		if m.statusErr {
			statusLine = statusErrStyle.Render(m.status)
		} else {
			statusLine = statusOkStyle.Render(m.status)
		}
	}

	var body, help string
	switch m.activeTab {
	case tabIssues:
		body = m.issueList.View()
		issuesHelp := "enter: checkout as worktree  •  o: open in browser  •  /: search  •  a: mine/all"
		if len(m.issueSources) > 1 {
			issuesHelp += "  •  s: switch issue repo"
		}
		issuesHelp += "  •  tab: switch view  •  r: refresh  •  q/esc: quit"
		help = footerStyle.Render(issuesHelp)
	case tabPRs:
		body = m.prList.View()
		help = footerStyle.Render("enter: checkout as worktree  •  o: open in browser  •  /: search  •  a: mine/all  •  tab: switch view  •  r: refresh  •  q/esc: quit")
	default:
		body = m.worktreeList.View()
		help = footerStyle.Render("enter: open in herdr  •  c: select & cd  •  o: open in browser  •  n: new  •  d: delete  •  l: lock  •  R: rename  •  p: prune  •  tab: switch view  •  r: refresh  •  q/esc: quit")
	}

	if m.searching {
		help = footerStyle.Render(m.search.View() + "  •  enter: search  •  esc: cancel")
	}

	return titleStyle.Render("lazyworktree") + "  " + tabBar + "\n" + body + "\n" + statusLine + "\n" + help
}
